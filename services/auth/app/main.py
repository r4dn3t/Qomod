import asyncio
from contextlib import asynccontextmanager
import logging

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.config import settings
from app.consumer import start_consumer
from app.routers import auth

logging.basicConfig(
    level=settings.LOG_LEVEL,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s"
)
logger = logging.getLogger("auth.main")


@asynccontextmanager
async def lifespan(app: FastAPI):
    # 1. Startup: Launch background RabbitMQ consumer
    stop_event = asyncio.Event()
    consumer_task = asyncio.create_task(start_consumer(stop_event))
    logger.info("Auth microservice initialized on port %s", settings.PORT)

    yield

    # 2. Shutdown: Cancel consumer and wait for clean exit
    logger.info("Shutting down auth microservice...")
    stop_event.set()
    consumer_task.cancel()
    try:
        await asyncio.wait_for(consumer_task, timeout=5.0)
    except (asyncio.CancelledError, asyncio.TimeoutError):
        pass
    logger.info("Auth microservice stopped cleanly.")


app = FastAPI(
    title="Qomod Auth Microservice",
    version="1.0.0",
    description="Authentication, JWT token issuance, session tracking, and real-time event-driven revocation",
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(auth.router)


@app.get("/healthz")
async def health_check():
    return {"status": "ok", "service": "auth"}
