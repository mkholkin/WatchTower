ARG BASE=watchtower-lab4-app:local
FROM ${BASE}
WORKDIR /workspace
COPY go.mod go.sum ./
COPY cmd ./cmd
COPY pkg ./pkg
COPY api ./api
COPY internal ./internal
COPY configs ./configs
COPY migrations ./migrations
COPY tests/lab4 ./tests/lab4
COPY tools/lab4 ./tools/lab4
COPY tools/lab8 ./tools/lab8
RUN go build -v -o /usr/local/bin/watchtower-api ./cmd/api
ENTRYPOINT ["python3"]
