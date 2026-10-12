| Требование | Тест, файл или команда |
|---|---|
| Пользовательский сценарий с входом, действиями и паузами | `tests/lab4/journey.js` |
| Достаточное число наблюдений | `tools/lab4/report.py`: ≥ 1000 запросов на полный профиль |
| Постепенное увеличение нагрузки | `python3 tools/lab4/run.py --profile ramp` |
| Перегрузка и восстановление | `python3 tools/lab4/run.py --profile stress` |
| Длительная стабильная нагрузка | `python3 tools/lab4/run.py --profile soak-0` |
| 10% и 20% ошибочных действий | `tests/lab4/model.test.mjs`, профили `soak-10`, `soak-20` |
| Отдельные образы и одинаковая конфигурация | `tools/lab4/run.py`, `tools/lab4/compose.yaml`, `manifest.json` |
| Временной ряд, перцентили, гистограмма задержек | `tools/lab4/report.py`, `profiles/*/plots/latency.png` |
| RPS и его распределение | `profiles/*/plots/throughput.png`, `summary.json` |
| CPU, RAM, диск каждого компонента; min/max/median | `tools/lab4/collect.py`, `profiles/*/plots/resources.png` |
| Отдельные CPU системы, генератора и мониторинга | `tools/lab4/compose.yaml`, `profiles/*/inspect.json` |
| Проверка расчётов | `make lab4-check` |
| Проверка достоверности времени | `tools/lab4/collect.py`, `test_report.py`: wall/monotonic, отрицательные длительности |
| Allure, живые логи и CI-артефакты | `make lab4`, `.github/workflows/tiopo-lab4.yml` |
| Измеренные результаты и выводы | `tiopo/lab4/results.md`, `tiopo/lab4/analysis.md` |
