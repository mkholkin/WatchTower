MOCKGEN_BIN := $(shell go env GOPATH)/bin/mockgen

.PHONY: test mocks ensure-mockgen

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


