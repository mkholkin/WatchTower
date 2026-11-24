| Требование | Файл или команда |
|---|---|
| Использовать проект и пять профилей ЛР №4 | `tools/lab6/run.py`; `tests/lab4/model.mjs`; `make lab6` |
| Интегрировать OpenTelemetry для отладки | `internal/observability/trace.go`; Go SDK и OTLP/HTTP exporter |
| Связать HTTP и SQL spans | Gin middleware; `pgx.QueryTracer`; `X-Trace-ID`; W3C Trace Context |
| Сравнить четыре настройки трассировки и логирования | `baseline`, `trace`, `debug`, `trace-debug`; 20 опытов |
| Доставлять трассы в Tempo во время нагрузки | `tools/lab6/otel-collector.yaml`; `tempo-experiment.yaml`; `tempo-live-check.json` |
| Сохранять трассы в CI/CD | Collector file exporter; `traces-otlp.jsonl`; `.github/workflows/tiopo-lab6.yml` |
| Сравнить CPU и память приложения с трассировкой и без неё | `tools/lab6/report.py`: `resource_summary`, `compare`; `results.md` |
| Сравнить обычные и расширенные логи | `WT_HTTP_DEBUG`; `logging.level`; режимы `debug`, `trace-debug` |
| Отдельно оценить средства мониторинга и трассировки | Компоненты `collector`, `otel-collector`, `tempo` в `resources.jsonl`; сводки CPU/RAM |
| Проверить 100% выборку и завершение пакетного экспорта | `export-summary.json`; `otel-collector-metrics.txt`; `export_integrity` |
| Проверить отсутствие секретов в трассах и логах | `internal/observability/trace_test.go`; `tools/lab6/test_report.py` |
| Проверить parent IDs и длительности HTTP → SQL | `tools/lab6/report.py`: `read_otlp`; `tempo-live-trace.json` |
| Сформировать сравнимый отчёт | Allure, `results.md`, `summary.json`, `plots/`; `tools/lab6/report.py` |
| Показать метрики и трассы в Grafana | `make lab6-grafana`; `tools/lab6/grafana/`; `tiopo/lab6/grafana.md` |
| Измерить стоимость просмотра | `make lab6-grafana-check`; `viewer-resources.json`, `viewer-resources.jsonl` |
