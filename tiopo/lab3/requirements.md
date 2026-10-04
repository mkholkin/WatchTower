| Требование | Тестовый файл или команда |
|---|---|
| Два объекта сравнения | `benchmarks/lab3/main.go`: `encoding/json` и `goccy/go-json` |
| 1. Не менее 100 испытаний и статистика | `make lab3`; `tools/lab3/report.py`, `test_report.py` |
| 2. Warmup и teardown | `benchmarks/lab3/main.go`, `main_test.go`; `raw/trial-*.json` |
| 3. Инструмент микробенчмаркинга | `benchmarks/lab3/main.go`: пакетный таймер и отдельные задержки |
| 4. Одно ядро CPU | `tools/lab3/run.py`: CPU 0, 1 CPU, GOMAXPROCS=1; `manifest.json` |
| 5. Несколько сценариев | `encode`, `decode`, `roundtrip`, массивы 1 и 32; `main_test.go` |
| 6. Отдельные образы и равная конфигурация | `tools/lab3/Dockerfile`, `run.py`; `manifest.json` |
| 7. Overhead инструмента | `control`, `measure` в `main.go`; `summary.json`: `overhead_ns_per_op`, `timer_pair_ns_per_op` |
| 8a. График во времени | `plots/*.svg`, `plots/*.png`: Time series |
| 8b. Распределение по доле запросов | `plots/*.svg`, `plots/*.png`: Empirical CDF |
| 8c. Гистограмма | `plots/*.svg`, `plots/*.png`: Latency histogram |
| 8d. p50, p75, p90, p95, p99 | `summary.csv`, `summary.json`; `test_report.py` |
| 8e. Время теста | `duration_ns` в JSON, `manifest.json`: `campaign_seconds` |
| 8f. Число операций, warmup и teardown | `summary.csv`, `raw/trial-*.json`; `main_test.go` |
| Allure и GitHub CI | `.github/workflows/tiopo-lab3.yml`; `allure-report/index.html` |
