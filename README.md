# TPC Online Checker

A Go-based application that monitors the VATSIM datafeed to track online status for members of The Pilot Club (TPC). It sends real-time notifications via Discord webhooks when pilots start a flight, update their flight plans, or when ATC becomes active.

## Overview

The TPC Online Checker consists of two main components:
- **Pilot Checker**: Monitors VATSIM for TPC members who are flying.
- **ATC Checker**: Monitors VATSIM for TPC members who are providing Air Traffic Control services.

The application uses Redis to store session state, ensuring that notifications are only sent once per session and updated appropriately if flight plans change.

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

To run the Pilot Checker:
```bash
go run main.go
```

To run the ATC Checker:
```bash
go run atc-main.go
```

### Running with Docker

The provided `Dockerfile` uses a multi-stage build to compile both bots.

1. Build the image:
   ```bash
   docker build -t tpc-online-checker .
   ```

2. Run the Pilot Checker:
   ```bash
   docker run --env-file .env tpc-online-checker ./bot
   ```

3. Run the ATC Checker:
   ```bash
   docker run --env-file .env tpc-online-checker ./bot-atc
   ```

## Environment Variables

Create a `.env` file or set the following variables in your environment:

| Variable | Description | Example |
|----------|-------------|---------|
| `REDIS_URL` | Redis connection string. Use `rediss://` for TLS. | `redis://localhost:6379/0` |
| `REDIS_DB` | Optional. Redis database number; overrides the one in `REDIS_URL` when set. | `0` |
| `WEBHOOK_ID` | The ID of the Discord Webhook. | `123456789012345678` |
| `WEBHOOK_TOKEN` | The Token of the Discord Webhook. | `your-webhook-token-here` |

## Project Structure

- `main.go`: Entry point for the Pilot Checker.
- `atc-main.go`: Entry point for the ATC Checker.
- `functions/`: Contains the core logic for checking online status.
  - `online-checker.go`: Logic for pilot tracking.
  - `atc-online-checker.go`: Logic for ATC tracking.
- `Dockerfile`: Multi-stage Docker build configuration.
- `go.mod`: Go module definition and dependencies.

## Tests

- [ ] TODO: Implement unit and integration tests for core logic.

## License

- [ ] TODO: Specify license information (e.g., MIT, Apache 2.0).

---
*Made by the TPC Tech Team*
