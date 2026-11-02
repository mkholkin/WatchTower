FROM node:24-bookworm-slim AS node
FROM golang:1.26.3-bookworm
COPY --from=node /usr/local/bin/node /usr/local/bin/node
COPY --from=node /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -s /usr/local/lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm && \
    apt-get update && apt-get install -y --no-install-recommends python3-venv curl && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /workspace
COPY go.mod go.sum ./
RUN --mount=type=cache,id=watchtower-lab5-module-download,target=/module-download \
    GOPROXY=file:///module-download,https://proxy.golang.org,direct go mod download
COPY tools/lab1/package*.json tools/lab1/
RUN npm ci --prefix tools/lab1 --no-audit --no-fund
COPY tools/lab5/requirements.txt tools/lab5/
RUN python3 -m venv /opt/bdd && \
    /opt/bdd/bin/pip install --no-cache-dir -r tools/lab5/requirements.txt
ENV PATH="/opt/bdd/bin:${PATH}"
COPY . .
RUN go build -o /usr/local/bin/watchtower-api ./cmd/api
ENV GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOTELEMETRY=off
ENTRYPOINT ["python3", "tools/lab5/entrypoint.py"]
