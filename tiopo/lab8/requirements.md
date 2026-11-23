| Требование | Файл или команда |
|---|---|
| Проект из ЛР №1, 2 или 4 | Go-приложение WatchTower и сценарий `tests/lab4/journey.js` |
| Внешний сервис с описанным контрактом | Verisign RDAP; `api/contracts/rdap/README.md`; RFC 9082/9083 |
| Реализовать интеграцию в приложении | `internal/infra/rdap/client.go`; `internal/api/http/v1/handler/rdap.go` |
| Описать HTTP endpoint приложения | `api/rdap.openapi.yaml`; `/monitors/{monitorId}/registration` |
| Реализовать E2E интеграции | `tools/lab8/e2e.py`; `make lab8-mock`, `make lab8-real` |
| Реализовать mock-сервер | `tools/lab8/rdap_mock.py`; `api/contracts/rdap/fixtures/example.com.json` |
| Переключать mock и реальный сервис конфигурацией | `configs.RDAPConfig`; `rdap.base_url`, `rdap.timeout_ms` |
| Настроить CI с обоими сервисами | `.github/workflows/tiopo-lab8.yml`; `make lab8` |
| Запускать демонстрационный контур с mock | `make lab8-demo-mock`; `python3 tools/lab8/demo_request.py` |
| Запускать демонстрационный контур с реальным сервисом | `make lab8-demo-real`; `python3 tools/lab8/demo_request.py` |
| Провести фиксированную нагрузку ЛР №4 через mock | `make lab8-load`; профиль `soak-0`: 15 с прогрева и 300 с при 4 VU |
| Подтвердить обращения к внешней зависимости | `rdap-mock-stats.json`; проверка `validate_load` в `tools/lab8/run.py` |
| Сохранять Allure и исходные данные при сбое | `tools/lab8/report.py`; CI `if: always()` |
| Согласовать выбор сервиса с преподавателем | Verisign RDAP для доменов `.com`; выбор и ссылки в `tiopo/lab8/README.md` |
