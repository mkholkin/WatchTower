FROM node:24-bookworm-slim AS node
FROM golang:1.26.3-bookworm
COPY --from=node /usr/local/bin/node /usr/local/bin/node
COPY --from=node /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -s /usr/local/lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm && \
    apt-get update && apt-get install -y --no-install-recommends python3 curl tcpdump postgresql-client redis-tools && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY tools/lab1/package*.json tools/lab1/
RUN npm ci --prefix tools/lab1 --no-audit --no-fund
COPY . .
RUN go build -o /usr/local/bin/watchtower-api ./cmd/api && \
    go build -o /usr/local/bin/lab2-migrate ./tools/lab2/migrate
ENV GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOTELEMETRY=off
ENTRYPOINT ["python3", "tools/lab2/pipeline.py"]
