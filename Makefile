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

test: test-unit

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

# Lab 1: setup is explicit; ordinary test runs never regenerate tracked mocks.
.PHONY: lab1-prepare test-unit test-unit-shuffle test-unit-race test-unit-offline lab1-offline-image test-integration coverage coverage-branch allure-report allure-open test-processes lab1
lab1-prepare:
	./tools/lab1/run.sh prepare

test-unit:
	./tools/lab1/run.sh unit

test-unit-shuffle:
	SEED=$(or $(SEED),42) ./tools/lab1/run.sh shuffle

test-unit-race:
	./tools/lab1/run.sh race

coverage:
	./tools/lab1/run.sh coverage

coverage-branch:
	./tools/lab1/run.sh branch

allure-report:
	./tools/lab1/run.sh report

allure-open:
	./tools/lab1/run.sh open

test-processes:
	./tools/lab1/run.sh processes

lab1:
	./tools/lab1/run.sh all

lab1-offline-image:
	docker build -f tools/lab1/Dockerfile -t watchtower-lab1:local .

test-unit-offline:
	@mkdir -p build/lab1/offline
	docker run --rm --network none -v "$(CURDIR)/build/lab1/offline:/workspace/build/lab1" watchtower-lab1:local

test-integration:
	go test -tags=integration ./internal/infra/repository/postgres

# Lab 2: isolated real stores and ordered container pipeline.
.PHONY: lab2-build lab2
lab2-build:
	./tools/lab2/run.sh build

lab2:
	./tools/lab2/run.sh

# Lab 3: independent single-CPU JSON microbenchmarks and statistical reports.
.PHONY: lab3 lab3-smoke lab3-check
lab3:
	python3 tools/lab3/run.py

lab3-smoke:
	python3 tools/lab3/run.py --smoke --trials 1

lab3-check:
	go test ./benchmarks/lab3
	python3 -m unittest discover -s tools/lab3 -p 'test_*.py'

# Lab 4: real API load, recovery, resource monitoring and Allure.
.PHONY: lab4 lab4-smoke lab4-check
lab4:
	python3 tools/lab4/run.py

lab4-smoke:
	python3 tools/lab4/run.py --smoke

lab4-check:
	node --test tests/lab4/model.test.mjs
	python3 -m unittest discover -s tools/lab4 -p 'test_*.py'

# Lab 5: real HTTP/SMTP BDD authentication with isolated ephemeral stores.
.PHONY: lab5 lab5-check
lab5:
	python3 tools/lab5/run.py

lab5-check:
	python3 -m unittest discover -s tools/lab5 -p 'test_*.py'
	python3 -m unittest discover -s tests/lab5 -p 'test_*.py'

# Lab 6: unchanged Lab 4 workloads with tracing/logging in a 2x2 matrix.
.PHONY: lab6 lab6-smoke lab6-check
lab6:
	python3 tools/lab6/run.py

lab6-smoke:
	python3 tools/lab6/run.py --smoke

lab6-check:
	node --test tests/lab4/model.test.mjs
	python3 -m unittest discover -s tools/lab6 -p 'test_*.py'

.PHONY: lab6-grafana lab6-grafana-down lab6-grafana-check
LAB6_VIEW_SECONDS ?= 5
LAB6_RUN_DIR_ARG = $(if $(RUN_DIR),--run-dir "$(RUN_DIR)")

lab6-grafana:
	python3 tools/lab6/grafana_stack.py up $(LAB6_RUN_DIR_ARG) --measure-seconds $(LAB6_VIEW_SECONDS)

lab6-grafana-down:
	python3 tools/lab6/grafana_stack.py down $(LAB6_RUN_DIR_ARG)

lab6-grafana-check:
	python3 tools/lab6/grafana_stack.py check $(LAB6_RUN_DIR_ARG) --measure-seconds $(LAB6_VIEW_SECONDS)

# Lab 7: one static gate for the working tree, staged commits and CI.
.PHONY: lint install-hooks lab7-check
lint:
	python3 tools/lab7/check.py

install-hooks:
	git config --local core.hooksPath .githooks

lab7-check:
	GOMODCACHE="$(CURDIR)/.cache/lab7/modules" GOCACHE="$(CURDIR)/.cache/go-build" go test -C tools/lab7/metrics ./...
	python3 -m unittest discover -s tools/lab7 -p 'test_*.py'

# Lab 8: configurable external RDAP service, mock/real E2E and fixed mock load.
.PHONY: lab8 lab8-smoke lab8-check lab8-mock lab8-real lab8-load lab8-demo-mock lab8-demo-real lab8-down
lab8:
	python3 tools/lab8/run.py

lab8-smoke:
	python3 tools/lab8/run.py --smoke

lab8-mock:
	python3 tools/lab8/run.py e2e --mode mock

lab8-real:
	python3 tools/lab8/run.py e2e --mode real

lab8-load:
	python3 tools/lab8/run.py load

lab8-demo-mock:
	python3 tools/lab8/run.py demo --mode mock

lab8-demo-real:
	python3 tools/lab8/run.py demo --mode real

lab8-down:
	python3 tools/lab8/run.py down

lab8-check:
	GOCACHE="$(CURDIR)/.cache/go-build" go test ./internal/infra/rdap ./internal/api/http/v1/handler ./cmd/api ./configs
	python3 -m unittest discover -s tools/lab8 -p 'test_*.py'
	node --test tests/lab4/model.test.mjs tests/lab8/journey.test.mjs
