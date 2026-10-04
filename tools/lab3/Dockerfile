FROM golang:1.26.3-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY internal/api/http/v1/gen ./internal/api/http/v1/gen
COPY benchmarks/lab3 ./benchmarks/lab3
RUN CGO_ENABLED=0 go build -trimpath -o /benchmark ./benchmarks/lab3

FROM node:24-bookworm-slim AS node
FROM python:3.13-slim-bookworm
COPY --from=node /usr/local/bin/node /usr/local/bin/node
COPY --from=node /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -s /usr/local/lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm
WORKDIR /workspace
COPY tools/lab3/requirements.txt /tmp/requirements.txt
RUN pip install --no-cache-dir -r /tmp/requirements.txt
COPY tools/lab1/package*.json tools/lab1/
RUN npm ci --prefix tools/lab1 --no-audit --no-fund
COPY tools/lab3 tools/lab3/
COPY --from=build /benchmark /usr/local/bin/lab3-benchmark
ENV GOMAXPROCS=1 GOGC=100 GOMEMLIMIT=384MiB MPLCONFIGDIR=/tmp/matplotlib
ENTRYPOINT ["/usr/local/bin/lab3-benchmark"]
