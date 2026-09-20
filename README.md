# WatchTower

[![Go Version](https://img.shields.io/badge/Go-1.26.0-green)](https://github.com/golang/go/tree/go1.26.0)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**WatchTower** is a self-hosted web resource monitoring system.
---

## 📢 Screenshots

### Dashboard
![Dashboard](img/img1.png)


### Monitor Details
![Monitor Details](img/img2.png)
---

## ✨ Key Features

- **Target Deduplication:** Automatically merges identical targets across different users to reduce network and storage overhead.
- **Multi-user Support:** Built from the ground up for teams. Each user has their own monitors, notification channels, and maintenance windows.
- **Flexible Notifications:** Easily route alerts to Telegram, Email, and more.
- **Maintenance Windows:** Temporarily mute alerts and pause SLA calculations during planned downtimes.

---

## 😀 Getting Started

The easiest way to run the Web UI, API and infrastructure is Docker Compose.

### Prerequisites
- [Docker](https://docs.docker.com/get-docker/)

### Quick Start (Docker)

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/WatchTower.git
   cd WatchTower
   ```

2. Run application:
   ```bash
   docker compose up --build
   ```

3. Open the required interface:

   - Web UI: http://localhost:3000
   - Swagger UI: http://localhost:8080/swagger
   - OpenAPI source: http://localhost:8080/openapi.yaml
   - REST API v2: `http://localhost:8080/api/v2`

4. Run the terminal UI in another shell. It uses the generated API v2 client:

   ```bash
   go run ./cmd/cli http://localhost:8080
   ```

The legacy API remains available under `/api/v1`. Its unversioned routes are also
kept so the existing Web UI continues to work.

### API development

`api/openapi.yaml` is the source of truth for REST API v2. Regenerate the Go
server and client types after changing it, then validate the contract:

```bash
make gen-api
GOCACHE=/tmp/watchtower-go-cache make validate-api
go test ./internal/api/http/v2/... ./cmd/api ./cmd/cli
```

---
