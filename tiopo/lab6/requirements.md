| Требование | Файл или команда |
|---|---|
| Повторить пять профилей ЛР №4 | `tools/lab6/run.py`; `make lab6` |
| Трассировка и расширенные логи в матрице 2×2 | `tools/lab6/run.py`; `WT_TRACE_FILE`, `WT_HTTP_DEBUG` |
| Сохранить сырые трассы и метрики | `variants/*/repetitions/*/profiles/*/traces.jsonl`, `resources.jsonl` |
| CPU API и сборщика в CPU-секундах | `tools/lab6/report.py`: `resource_summary`; `summary.json` |
| RAM min/median/max и разница с baseline того же профиля | `tools/lab6/report.py`: `resource_summary`, `compare` |
| Учитывать RPS и ошибки исходного сценария | `tools/lab4/report.py`: `summarize`; `results.md`, `summary.json` |
| Отдельно оценить накладные расходы сборщика | `tools/lab6/report.py`: компонент `collector` |
| Проверить целостность ресурсов и наличие трасс | `tools/lab6/test_report.py`; `make lab6-check` |
| Исключить секреты из трасс | `tools/lab6/test_report.py`: `test_trace_secrets_and_raw_identifiers_rejected` |
| Allure, Markdown, JSON и графики | `tools/lab6/report.py`; `allure-results/`, `results.md`, `summary.json`, `plots/` |
| Сохранить артефакты опыта в CI | `.github/workflows/tiopo-lab6.yml` |
| Зафиксировать методику и ограничения | `tiopo/lab6/README.md` |
