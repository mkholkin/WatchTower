| Требование | Тестовый файл или команда |
|---|---|
| 1. Отдельный Docker-контейнер тестов, запуск из CI | [Dockerfile](../../tools/lab2/Dockerfile), [tiopo-lab2.yml](../../.github/workflows/tiopo-lab2.yml); `make lab2` |
| 2. Реальные хранилища | [fixture_test.go](../../tests/lab2/integration/fixture_test.go), [probe_metrics_test.go](../../tests/lab2/integration/probe_metrics_test.go) |
| 3. Отдельный экземпляр хранилища | [compose.yaml](../../tools/lab2/compose.yaml) |
| 4. Воспроизводимая подготовка данных | [migrate/main.go](../../tools/lab2/migrate/main.go), [env.go](../../tests/lab2/testsupport/env.go) |
| 5. Порядок unit → integration → e2e | [pipeline.py](../../tools/lab2/pipeline.py); `make lab2` |
| 6. E2E на развёрнутом стенде | [monitor_http_test.go](../../tests/lab2/e2e/monitor_http_test.go) |
| 7. Мобильное устройство | Не применимо: HTTP API, мобильного приложения нет |
| 8. Остановка этапов при ошибке, skipped и обязательный отчёт | `LAB2_FAIL_STAGE=integration make lab2`; [pipeline.py](../../tools/lab2/pipeline.py) |
| 9. Тренды запусков | [allurerc.mjs](../../tools/lab2/allurerc.mjs); `make lab2` дважды |
| 10. Восстановление после ошибки и прерывания | `LAB2_FAIL_STAGE=e2e make lab2`; [pipeline.py](../../tools/lab2/pipeline.py), [run.sh](../../tools/lab2/run.sh) |
| 11. Очистка очереди сообщений | [fixture_test.go](../../tests/lab2/integration/fixture_test.go), [pipeline.py](../../tools/lab2/pipeline.py) |
| 12. Завершение пользовательских сессий | [pipeline.py](../../tools/lab2/pipeline.py): остановка API и удаление одноразового JWT-ключа |
| 13. E2E без GUI | [monitor_http_test.go](../../tests/lab2/e2e/monitor_http_test.go) |
| 14. Повторяемость интеграционных тестов | `LAB2_INTEGRATION_REPEAT=2 make lab2` |
| 15. Независимость параллельных запусков | `make lab2 & make lab2 & wait`; [run.sh](../../tools/lab2/run.sh) |
| 16. Успешное прохождение | `make lab2`; [results.md](results.md) |
| 17. Отдельные файлы сценариев, одна подготовка, тестовые суффиксы | [integration](../../tests/lab2/integration): `*_test.go`, тег `lab2integration` |
| Имитация E2E запросами и лог трафика | [replay.py](../../tools/lab2/replay.py); `curl-replay.log`, `traffic.pcap`, `traffic.txt` после `make lab2` |
