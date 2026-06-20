# TPC Online Checker

A Go-based application that monitors the VATSIM datafeed to track online status for members of The Pilot Club (TPC). It sends real-time notifications via Discord webhooks when pilots start a flight, update their flight plans, or when ATC becomes active.

## Overview

The TPC Online Checker ships as a single `presence-checker` binary that runs two checks:
- **Pilot Checker**: Monitors VATSIM for TPC members who are flying.
- **ATC Checker**: Monitors VATSIM for TPC members who are providing Air Traffic Control services.

Each check can be enabled or disabled independently via the `ENABLE_PILOT` and `ENABLE_ATC` environment variables (both default to enabled). The application uses Redis to store session state, ensuring that notifications are only sent once per session and updated appropriately if flight plans change.

## Requirements

- **Go**: 1.26.0 or higher
- **Redis**: A running Redis instance for state management.
- **Discord Webhook**: A webhook URL (split into ID and Token) for sending notifications.
- **TPC API Access**: The application uses the `tpcgo` library to interact with The Pilot Club's internal services.

## Setup & Run

### Local Environment Setup

1. Clone the repository.
2. Create a `.env` file in the root directory (see [Environment Variables](#environment-variables)).
3. Install dependencies:
   ```bash
   go mod download
   ```

### Running with Go

To run both checks:
```bash
go run ./cmd/presence-checker
```

To run only one check, disable the other via environment variable:
```bash
ENABLE_ATC=false go run ./cmd/presence-checker   # pilot only
ENABLE_PILOT=false go run ./cmd/presence-checker # ATC only
```

### Running with Docker

The provided `Dockerfile` uses a multi-stage build to compile the binary.

1. Build the image:
   ```bash
   docker build -t tpc-online-checker .
   ```

2. Run it (both checks enabled by default; toggle with `ENABLE_ATC` / `ENABLE_PILOT`):
   ```bash
   docker run --env-file .env tpc-online-checker
   ```

## Environment Variables

Create a `.env` file or set the following variables in your environment:

| Variable | Description | Example |
|----------|-------------|---------|
| `REDIS_URL` | Redis connection string. Use `rediss://` for TLS. | `redis://localhost:6379/0` |
| `REDIS_DB` | Optional. Redis database number; overrides the one in `REDIS_URL` when set. | `0` |
| `WEBHOOK_ID` | The ID of the Discord Webhook. | `123456789012345678` |
| `WEBHOOK_TOKEN` | The Token of the Discord Webhook. | `your-webhook-token-here` |
| `ENABLE_PILOT` | Optional. Enable the pilot check. Defaults to `true`. | `true` |
| `ENABLE_ATC` | Optional. Enable the ATC check. Defaults to `true`. | `true` |
| `HEALTH_ADDR` | Optional. Listen address for the health endpoints. Defaults to `:8080`. | `:8080` |

## Health & Kubernetes

The binary serves two endpoints on `HEALTH_ADDR` for use as Kubernetes probes:

- `GET /health` (liveness): `200` while every enabled check is iterating; `503` if a check has not completed an iteration within ~4× the poll interval (i.e. the loop has wedged).
- `GET /ready` (readiness): `200` once start-up wiring is complete, `503` before that and during shutdown.

On `SIGTERM`/`SIGINT` the checkers finish their current pass, stop, and the process exits cleanly, so it terminates within the default grace period.

> **Run a single replica.** Notification de-duplication relies on a check-then-write against Redis with no leader election, so two replicas sharing one Redis can race and post duplicate notifications. Use `replicas: 1` with the `Recreate` update strategy.

## Project Structure

- `cmd/presence-checker/`: Entry point; runs the enabled checks concurrently.
- `internal/functions/`: Core logic for checking online status.
  - `tracker.go`: Generic reconciliation engine shared by both checks.
  - `discord.go`: Discord webhook client and embed helpers.
  - `online-checker.go`: Pilot tracking handler.
  - `atc-online-checker.go`: ATC tracking handler.
- `internal/store/`: Redis-backed session state store.
- `Dockerfile`: Multi-stage Docker build configuration.
- `go.mod`: Go module definition and dependencies.

## Tests

Run the unit tests for the core reconciliation logic:
```bash
go test ./...
```

## License

- [ ] TODO: Specify license information (e.g., MIT, Apache 2.0).

---
*Made by the TPC Tech Team*
