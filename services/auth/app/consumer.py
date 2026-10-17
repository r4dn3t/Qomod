import asyncio
import json
import logging
from typing import Any, Dict

import aio_pika

from app.config import settings
from app.session_store import session_store

logger = logging.getLogger("auth.consumer")


async def process_event(routing_key: str, payload: Dict[str, Any]) -> None:
    """Processes events from users service to manage session invalidation."""
    aggregate_id = payload.get("aggregate_id")
    event_type = payload.get("event_type")
    data = payload.get("data", {})

    logger.info("Received event [%s] for aggregate [%s]", routing_key, aggregate_id)

    if not aggregate_id:
        return

    if routing_key == "user.password.changed":
        logger.warning("Revoking all active sessions for user [%s] due to password change", aggregate_id)
        await session_store.revoke_all_user_sessions(aggregate_id)

    elif routing_key == "user.status.changed":
        new_status = data.get("new_status")
        if new_status and new_status != "enabled":
            logger.warning("Revoking sessions for user [%s] due to account status '%s'", aggregate_id, new_status)
            await session_store.revoke_all_user_sessions(aggregate_id)

    elif routing_key == "user.deleted":
        logger.warning("Revoking sessions for user [%s] due to account deletion", aggregate_id)
        await session_store.revoke_all_user_sessions(aggregate_id)


async def start_consumer(stop_event: asyncio.Event) -> None:
    """Background worker consuming user events with auto-reconnect."""
    while not stop_event.is_set():
        try:
            logger.info("Connecting to RabbitMQ at %s...", settings.RABBITMQ_URL)
            connection = await aio_pika.connect_robust(settings.RABBITMQ_URL)

            async with connection:
                channel = await connection.channel()
                await channel.set_qos(prefetch_count=10)

                exchange = await channel.declare_exchange(
                    settings.RABBITMQ_EXCHANGE,
                    aio_pika.ExchangeType.TOPIC,
                    durable=True,
                )

                queue = await channel.declare_queue(
                    settings.RABBITMQ_QUEUE,
                    durable=True,
                )

                for rk in ["user.password.changed", "user.status.changed", "user.deleted"]:
                    await queue.bind(exchange, routing_key=rk)

                logger.info("Auth service subscribed to queue '%s'", settings.RABBITMQ_QUEUE)

                async with queue.iterator() as queue_iter:
                    async for message in queue_iter:
                        if stop_event.is_set():
                            break

                        async with message.process():
                            try:
                                body = message.body.decode("utf-8")
                                event_payload = json.loads(body)
                                await process_event(message.routing_key, event_payload)
                            except Exception as ex:
                                logger.error("Error processing event: %s", ex, exc_info=True)

        except asyncio.CancelledError:
            break
        except Exception as ex:
            logger.warning("RabbitMQ consumer error: %s. Reconnecting in 5s...", ex)
            try:
                await asyncio.wait_for(stop_event.wait(), timeout=5.0)
            except asyncio.TimeoutError:
                pass
