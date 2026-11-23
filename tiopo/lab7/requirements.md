| Требование | Файл или команда |
|---|---|
| Вычислять цикломатическую сложность | `tools/lab7/metrics/main.go`; `make lint` |
| Максимальная сложность каждой функции и замыкания ≤10 | `-max-cyclomatic 10`; `tiopo/lab7/metrics.json` |
| Исправить существующий код сверх порога | Рефакторинг `cmd/`, `benchmarks/`, `internal/`, `tests/lab2/`; `tiopo/lab7/results.md` |
| Вычислять метрики Холстеда | `tools/lab7/metrics/main.go`: `measure`; `tiopo/lab7/metrics.json` |
| Проверять принятый в Go стиль | `gofmt -l` в `tools/lab7/check.py` |
| Линтер и проверка типов | `go vet -mod=readonly` с тегами интеграционных тестов в `tools/lab7/check.py` |
| Блокировать коммит при провале проверок | `make install-hooks`; `.githooks/pre-commit`; `tools/lab7/check.py --staged` |
| Проверять подготовленный к коммиту код, включая `git commit -a` | `tools/lab7/test_gate.py`; `make lab7-check` |
| Разрешить явный обход через `--no-verify` | Стандартный `git commit --no-verify`; `test_no_verify_bypasses_and_valid_commit_recovers` |
| Выполнять статический анализ в CI перед тестами | `.github/workflows/static-analysis.yml`; `needs: static` в workflow ЛР №2–7 |
| Сохранить метрики и проверить границу 10/11 | Артефакт `static-analysis-*`; `tools/lab7/metrics/main_test.go` |
