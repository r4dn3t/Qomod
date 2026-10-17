# RabbitMQ Complete Guide: From Basics to Event-Driven Architecture

This guide covers the core concepts, setup, management, and best practices for building an event-driven microservices architecture using RabbitMQ.

---

## 1. Core Architecture & Concepts

In RabbitMQ, producers never send messages directly to queues. Every message travels through an explicit pipeline:
* **Producer:** The application creating and publishing the payload.
* **Exchange:** The router that accepts messages and inspects routing keys/headers to determine destination queues.
* **Binding & Routing Key:** The binding links an exchange to a queue. The routing key is the address label attached to the message.
* **Queue:** An in-memory or disk-backed FIFO buffer holding messages until delivery.
* **Consumer:** The worker subscribing to a queue to process messages.

### Exchange Types
* **Direct:** Exact routing key match (e.g., `orders.created` to `orders.created`).
* **Fanout:** Broadcasts to all bound queues, ignoring the routing key.
* **Topic:** Wildcard pattern matching (`*` matches one word, `#` matches zero or more).
* **Headers:** Routes based on key-value metadata in message headers.

---

## 2. Installation & Setup (Docker)

Running RabbitMQ via Docker isolates the environment and makes management easy.

```bash
docker run -d \
  --name rabbitmq-server \
  --restart unless-stopped \
  -p 5672:5672 \
  -p 15672:15672 \
  -v rabbitmq_data:/var/lib/rabbitmq \
  rabbitmq:3-management
```
* **Port 5672:** Native AMQP port for application connections.
* **Port 15672:** Management HTTP Web UI / REST API.

---

## 3. Virtual Hosts (Vhosts) & Security

A **vhost** is an isolated sandbox within the RabbitMQ server. It has its own independent exchanges, queues, and permissions. It is used for multi-tenancy (multiple apps on one server) and environment separation (dev/test/prod).

### Managing Vhosts and Users (`rabbitmqctl`)
Execute these inside the container (`docker exec -it rabbitmq-server bash`):

```bash
# Create a new vhost
rabbitmqctl add_vhost /ecommerce_app

# Create a user
rabbitmqctl add_user ecommerce_admin SuperSecret!

# Assign Management Tag (Required for Web UI & rabbitmqadmin)
rabbitmqctl set_user_tags ecommerce_admin management

# Grant Permissions (Configure, Write, Read)
rabbitmqctl set_permissions -p /ecommerce_app ecommerce_admin ".*" ".*" ".*"
```

**Understanding Tags:**
Tags dictate HTTP API/Web UI access. 
* `None`: Can only connect via AMQP (code).
* `management`: Can log into the UI and manage assigned vhosts.
* `administrator`: Super-user with global access.

---

## 4. Event-Driven Architecture (Pub/Sub)

**Best Practice:** Services should emit *Domain Events* (historical facts, e.g., `user.registered`) to a shared **Topic Exchange**, rather than sending explicit *Commands* (e.g., `send_email`) directly to another service's queue. This perfectly decouples your microservices.

### Microservices Scenario Design:
* **Shared Vhost:** `/ecommerce_app`
* **Exchanges (Topic):** `user.events`, `wallet.events`, `auth.events`
* **Queues (One per action):** 
  * `q.wallet.create_deposit`
  * `q.message.all_notifications`
* **Bindings (The routing rules):**
  * `q.wallet.create_deposit` binds to `user.events` with key `user.registered`.
  * `q.message.all_notifications` binds to `user.events`, `wallet.events`, and `auth.events` using the wildcard `#` to catch *everything*.

---

## 5. Managing Topology via CLI (`rabbitmqadmin`)

While `rabbitmqctl` manages the server and vhosts, `rabbitmqadmin` (which uses the HTTP REST API) is used to create exchanges, queues, and bindings.

*Requires the `management` or `administrator` tag on the user.*

```bash
# Declare a Topic Exchange
rabbitmqadmin -V /ecommerce_app -u ecommerce_admin -p SuperSecret! declare exchange name=user.events type=topic

# Declare a Durable Queue
rabbitmqadmin -V /ecommerce_app -u ecommerce_admin -p SuperSecret! declare queue name=q.wallet.create_deposit durable=true

# Bind the Queue to the Exchange
rabbitmqadmin -V /ecommerce_app -u ecommerce_admin -p SuperSecret! declare binding source=user.events destination=q.wallet.create_deposit routing_key=user.registered
```

### Investigating and Debugging

```bash
# List all exchanges
rabbitmqadmin -V /ecommerce_app -u <user> -p <pass> list exchanges name type

# List all queues with memory and consumer stats
rabbitmqadmin -V /ecommerce_app -u <user> -p <pass> list queues name messages consumers memory

# List all routing bindings
rabbitmqadmin -V /ecommerce_app -u <user> -p <pass> list bindings source destination routing_key

# Safely peek at messages in a queue (requesue=true prevents data loss)
rabbitmqadmin -V /ecommerce_app -u <user> -p <pass> get queue=q.wallet.create_deposit count=5 ackmode=ack_requeue_true
```