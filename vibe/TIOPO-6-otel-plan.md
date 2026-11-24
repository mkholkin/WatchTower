# Переход ЛР №6 на OpenTelemetry

Заменить собственный recorder на OpenTelemetry Go SDK: 100% spans, пакетный OTLP/HTTP экспорт в OpenTelemetry Collector. Collector пишет OTLP JSONL в артефакты и передаёт трассы в Tempo во время нагрузки. Сохранить безопасные HTTP/SQL атрибуты, четыре режима и пять исходных профилей. Измерять API, Docker collector, OTel Collector и Tempo отдельно.

1. Заменить `internal/observability/trace.go`, подключение в `cmd/api/main.go` и `internal/app/app.go`; добавить SDK/OTLP зависимости и graceful shutdown. Проверить HTTP → SQL, propagation, отсутствие секретов, flush и ошибки экспортёра: `go test -race ./internal/observability ./cmd/api`.
2. Обновить `tools/lab6/run.py`, Compose, Dockerfile и конфиги Collector/Tempo: live OTLP, файловый экспорт `traces-otlp.jsonl`, корректная остановка API до Collector и Tempo, измерение дополнительных компонентов. Проверить `python3 -m unittest tools.lab6.test_run` и smoke всех четырёх режимов.
3. Обновить чтение OTLP, отчёты, импорт и дашборды Grafana; проверять parent IDs, безопасные атрибуты, полноту экспортированных spans. Сохранить нормализованный `traces.jsonl` как производный артефакт для Chrome/Grafana. Выполнить `make lab6-check` и проверку Grafana/Tempo.
4. Выполнить полную матрицу 20 опытов; обновить `tiopo/lab6` результатами, методикой, требованиями и стоимостью средств мониторинга. Проверить общий Go-набор, статический gate, сохранить изменения в TIOPO-6, перенести в TIOPO-7/TIOPO-8 и отправить три ветки. Остановить выделенный стенд после проверки.

## Уточнение пользователя 24.11.2026

Пользователь отменил ожидание часовой серии. Выполнение этапа 4 завершается по unit/race-тестам, smoke четырёх режимов, быстрой проверке Grafana и 11 завершённым нагрузочным опытам. Остальные опыты помечаются как невыполненные; полная матрица остаётся ручной командой и CI input `full_matrix=true`.
