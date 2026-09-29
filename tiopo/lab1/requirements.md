| Требование | Тестовый файл или команда |
|---|---|
| 1. Позитивные и негативные тесты публичных методов | [auth/service_test.go](../../internal/service/auth/service_test.go), [postgres/unit_test.go](../../internal/infra/repository/postgres/unit_test.go); [полная матрица](test-cases.md) |
| 2. Обработка ожидаемых ошибок | [service/error_test.go](../../internal/service/error_test.go), [auth/service_test.go](../../internal/service/auth/service_test.go) |
| 3. Классический и лондонский стили | [monitor/monitor_test.go](../../internal/domain/entity/monitor/monitor_test.go), [postgres/unit_test.go](../../internal/infra/repository/postgres/unit_test.go) |
| 4. Arrange–Act–Assert и fixtures | [redis/probe_summary_repository_test.go](../../internal/infra/repository/redis/probe_summary_repository_test.go) |
| 5. Одна публичная операция в Act | [alert_contact/contact_test.go](../../internal/domain/entity/alert_contact/contact_test.go), [maintenance/maintenance_test.go](../../internal/domain/entity/maintenance/maintenance_test.go) |
| 6. Проверка через публичные методы | [healthcheck/scheduler_test.go](../../internal/service/healthcheck/scheduler_test.go), [healthcheck/service_test.go](../../internal/service/healthcheck/service_test.go) |
| 7. Data Builder и Object Mother | [testutil/fixtures.go](../../internal/testutil/fixtures.go), [monitor/monitor_test.go](../../internal/domain/entity/monitor/monitor_test.go) |
| 8. Запуск из командной строки | `make lab1-prepare`, затем `make test-unit` |
| 9. Автоматический отчёт Allure | `make test-unit`, затем `make allure-report`; просмотр: `make allure-open` |
| 10. Случайный порядок тестов | `make test-unit-shuffle SEED=42` |
| 11. Запуск без интернета | `make lab1-offline-image`, затем `make test-unit-offline` |
| 12. Число тестовых процессов и настройка параллелизма | `make test-processes`; [описание PID, -p и -parallel](README.md#случайный-порядок-и-процессы) |
| 13. Успешное прохождение тестов | `make test-unit`, `make test-unit-race` |
| 14. Покрытие строк и ветвей | `make coverage`, `make coverage-branch` |
| 15. Защита от регрессий, устойчивость к рефакторингу | [monitoring_management/service_test.go](../../internal/service/monitoring_management/service_test.go), [healthcheck/scheduler_test.go](../../internal/service/healthcheck/scheduler_test.go) |
| Техники подготовки данных для каждого теста | [test-cases.md](test-cases.md); `make allure-report` |
