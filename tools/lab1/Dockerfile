FROM node:24-bookworm-slim AS node
FROM golang:1.26.3-bookworm
COPY --from=node /usr/local/bin/node /usr/local/bin/node
COPY --from=node /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -s /usr/local/lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm
RUN apt-get update && apt-get install -y --no-install-recommends python3 && rm -rf /var/lib/apt/lists/*
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY tools/lab1/package*.json tools/lab1/
RUN npm ci --prefix tools/lab1 --no-audit --no-fund
RUN GOBIN=/workspace/build/tools go install github.com/rillig/gobco@v1.3.4 && \
    GOBIN=/workspace/build/tools go install github.com/boumenot/gocover-cobertura@v1.5.0
COPY . .
ENV GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOTELEMETRY=off
# Compile unit packages once, without running tests or fetching images/services.
RUN xargs go test -run '^$' < tools/lab1/packages.txt
CMD ["bash", "-c", "tools/lab1/run.sh unit; status=$?; tools/lab1/run.sh report; report_status=$?; if [ $status -ne 0 ]; then exit $status; fi; exit $report_status"]
