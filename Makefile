MOCKGEN_BIN := $(shell go env GOPATH)/bin/mockgen
BUILD_DIR := build
APP_NAME := watchtower
BUSINESS_ARTIFACT := $(BUILD_DIR)/business.layer
DATA_ARTIFACT := $(BUILD_DIR)/data.layer
CLI_ARTIFACT := $(BUILD_DIR)/cli
APP_ARTIFACT := $(BUILD_DIR)/$(APP_NAME)
BUSINESS_PKGS := \
	./internal/service \
	./internal/service/analyze/... \
	./internal/service/auth/... \
	./internal/service/common/... \
	./internal/service/contacts/... \
	./internal/service/healthcheck/... \
	./internal/service/maintenance/... \
	./internal/service/metrics/... \
	./internal/service/monitoring_management/... \
	./internal/service/notification/...

.PHONY: test mocks ensure-mockgen build-business build-data build-cli build-components assemble-app build-app clean-build build gen-api gen-repo migrate-down benchmark-read benchmark-read-plot benchmark-update benchmark-update-plot benchmark-all

test: mocks
	go test ./...

mocks: ensure-mockgen
	$(MOCKGEN_BIN) -source=internal/domain/repo/user.go -destination=internal/service/testmocks/user_repo_mock.go -package=testmocks
	$(MOCKGEN_BIN) -source=internal/domain/repo/monitor.go -destination=internal/service/testmocks/monitor_repo_mock.go -package=testmocks
	$(MOCKGEN_BIN) -source=internal/domain/repo/target.go -destination=internal/service/testmocks/target_repo_mock.go -package=testmocks
	$(MOCKGEN_BIN) -source=internal/domain/repo/alert_contact.go -destination=internal/service/testmocks/alert_contact_repo_mock.go -package=testmocks
	$(MOCKGEN_BIN) -source=internal/domain/repo/maintenance_window.go -destination=internal/service/testmocks/maintenance_repo_mock.go -package=testmocks
	$(MOCKGEN_BIN) -source=internal/domain/repo/probe_result.go -destination=internal/service/testmocks/probe_result_repo_mock.go -package=testmocks
	$(MOCKGEN_BIN) -source=internal/domain/repo/probe_summary.go -destination=internal/service/testmocks/probe_summary_repo_mock.go -package=testmocks
	$(MOCKGEN_BIN) -source=internal/service/common/provider/user_provider.go -destination=internal/service/testmocks/user_provider_mock.go -package=testmocks
	$(MOCKGEN_BIN) -source=internal/service/testmocks/sources/probe_evaluator_source.go -destination=internal/service/testmocks/probe_evaluator_mock.go -package=testmocks
	$(MOCKGEN_BIN) -source=internal/service/testmocks/sources/healthcheck_prober_source.go -destination=internal/service/testmocks/healthcheck_prober_mock.go -package=testmocks
	$(MOCKGEN_BIN) -source=internal/service/testmocks/sources/subscriber_source.go -destination=internal/service/testmocks/subscriber_mock.go -package=testmocks

ensure-mockgen:
	@test -x "$(MOCKGEN_BIN)" || go install github.com/golang/mock/mockgen@v1.6.0

fmt:
	go fmt ./...
	swag fmt -d ./internal/api

gen-api:
	go tool oapi-codegen -config api/oapi-codegen.yaml api/openapi.yaml

gen-repo:
	sqlc generate

# Backward-compatible alias.
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(APP_ARTIFACT) ./cmd/cli

clean:
	rm -rf $(BUILD_DIR)

migrate:
	goose -dir migrations postgres "host=localhost port=5432 user=postgres password=postgres dbname=watchtower sslmode=disable" up

migrate-down:
	goose -dir migrations postgres "host=localhost port=5432 user=postgres password=postgres dbname=watchtower sslmode=disable" down

benchmark-read:
	go run ./benchmarks/read -x-start 1000000 -x-end 10000000 -x-step 1000000 -partitions 50 -iterations 10

benchmark-read-plot:
	python3 benchmarks/read/plot.py

benchmark-update:
	go run ./benchmarks/update -x-start 10000 -x-end 100000 -x-step 10000 -iterations 10

benchmark-update-plot:
	python3 benchmarks/update/plot.py

benchmark-all: benchmark-read benchmark-update
