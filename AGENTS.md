# AGENTS.md: Qomod Project Agent Context & Working Memory

> This document is automatically loaded by Antigravity IDE across all workstations.
> It maintains the working memory, architectural decisions, and current progress.

---

## 🧭 Project Identity & Overview

**Qomod** is a multi-stack, event-driven microservices workspace designed to learn and demonstrate best practices across **Go**, **.NET (C#)**, and **Python (FastAPI)**, communicating asynchronously via **RabbitMQ** (Topic Exchange).

---

## 🏛️ Microservices Roster

### 1. `users` Service
- **Runtime**: Golang 1.23
- **Port**: `8081`
- **Location**: `services/users`
- **Database**: PostgreSQL 16 (`qomod_users` on port `5432`)
- **Key Architecture**:
  - Clean Architecture / Hexagonal Ports & Adapters (`internal/domain`, `internal/usecase`, `internal/repository`, `internal/delivery`, `internal/event`).
  - Standard library `log/slog` structured logging, signal-driven graceful shutdown (`signal.NotifyContext`).
  - Connection pooling using `jackc/pgx/v5/pgxpool` with embedded schema migrations.
  - Resilient RabbitMQ publisher with persistent messages and auto-reconnection loop.
- **Data Model**:
  - `id` (UUID), `cellphone_number` (E.164, unique, mandatory), `first_name`, `last_name`, `birthdate` (YYYY-MM-DD), `email` (unique), `password_hash` (bcrypt), `status` (`enabled`, `disabled`, `suspended`, `deleted`), `phone_verified`, `email_verified`, `last_login_at`, timestamps, soft delete (`deleted_at`).
- **Events Emitted to `users.events`**:
  - `user.registered`
  - `user.profile.updated`
  - `user.password.changed`
  - `user.status.changed`
  - `user.deleted`

### 2. `notifications` Service
- **Runtime**: .NET 10 (C#)
- **Port**: `8082`
- **Location**: `services/notifications`
- **Key Architecture**:
  - ASP.NET Core Minimal APIs + `BackgroundService` hosted worker (`RabbitMqConsumerService`).
  - Consumes from RabbitMQ queue `notifications.user-events.queue` bound to `users.events`.
  - Dispatches SMS (phone) and Email notifications on `user.registered`, `user.password.changed`, `user.status.changed`, and `user.deleted`.
  - In-memory thread-safe audit store for dispatched notification records.

### 3. `auth` Service
- **Runtime**: Python 3.12+ (FastAPI)
- **Port**: `8083`
- **Location**: `services/auth`
- **Key Architecture**:
  - JWT token management (short-lived access tokens, long-lived refresh tokens with rotation).
  - Background `aio-pika` async consumer (`auth.user-events.queue`).
  - **Real-time Event-Driven Session Revocation**:
    - When `user.password.changed` or `user.status.changed` (suspended/disabled) or `user.deleted` occurs, the auth consumer immediately revokes all active refresh tokens and blacklists any access tokens issued prior to that timestamp.

---

## 🛠️ Development & Automation

- **Postgres & RabbitMQ**: `make dev-deps`
- **Run all tests**: `make test-all`
  - Go tests: `make test-users` (uses `-race`)
  - .NET tests: `make test-notifications` (`dotnet test`)
  - Python tests: `make test-auth` (`pytest`)
- **Run individual services**:
  - `make run-users` (port 8081)
  - `make run-notifications` (port 8082)
  - `make run-auth` (port 8083)
- **Docker Compose**: `docker compose up --build`
