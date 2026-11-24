# ЛР №6: стоимость OpenTelemetry и расширенных логов

WatchTower повторяет пять нагрузочных профилей ЛР №4 в четырёх режимах: `baseline`, `trace`, `debug`, `trace-debug`. В `trace` и `trace-debug` используется OpenTelemetry Go SDK с выборкой 100% spans и пакетным OTLP/HTTP экспортом. Расширенные HTTP-логи включаются отдельно.

## Трассировка

HTTP middleware создаёт server span, PostgreSQL hooks — дочерние client spans. OpenTelemetry управляет контекстом, идентификаторами и временем операций; W3C Trace Context обеспечивает передачу контекста. Ответ содержит `X-Trace-ID`. В spans сохраняются шаблон маршрута, HTTP-статус и тип SQL-команды. Тела запросов, заголовки авторизации, query string, SQL-текст и аргументы исключены.

Цепочка экспорта: **API → OpenTelemetry Collector → Tempo**. Collector одновременно записывает `traces-otlp.jsonl` в стандартном OTLP JSON для артефактов CI. Приложение завершает HTTP-запросы и экспорт до остановки Collector. Проверка полноты сравнивает число завершённых и отправленных spans SDK с уникальными spans в файловом экспорте. Поиск трассы в Tempo подтверждает живую доставку.

Для обычного запуска API:

```sh
WT_TRACING_ENABLED=true \
OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://localhost:4318/v1/traces \
OTEL_SERVICE_NAME=watchtower \
watchtower-api configs/local.yaml
```

Адрес должен указывать на запущенный OTLP/HTTP receiver. Без `WT_TRACING_ENABLED=true` SDK не создаётся. Расширенные HTTP-логи включаются `WT_HTTP_DEBUG=true` и `logging.level: debug`; запись содержит метод, шаблон маршрута, статус, длительность, размер ответа и trace ID при включённой трассировке.

Официальная документация: [Go SDK](https://opentelemetry.io/docs/languages/go/instrumentation/), [OTLP exporters](https://opentelemetry.io/docs/languages/go/exporters/), [Collector file exporter](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.140.0/exporter/fileexporter).

## Матрица и измерения

`baseline` — обычные логи; `trace` — трассировка; `debug` — расширенные логи; `trace-debug` — обе функции. Во всех режимах Collector и Tempo работают с одинаковыми лимитами. Это позволяет сравнить расходы при одинаковом составе стенда.

Профили: `ramp`, `stress`, `soak-0`, `soak-10`, `soak-20`. Каждый soak содержит 15 секунд прогрева и 300 секунд при четырёх пользователях. Исходный HTTP-сценарий ЛР №4 сохранён, MFA в одноразовом API выключена. Намеренные ошибочные операции 401/400 учитываются отдельно от неожиданных ответов.

CPU рассчитывается в CPU-секундах по приращению накопленного Docker-счётчика между границами измерительного окна, с линейной интерполяцией соседних отсчётов. RAM — working set без inactive file cache, для которого приводятся min/median/max. Прогрев и завершение экспорта исключены из нагрузочного окна.

Отдельно показываются API, Docker-сборщик метрик (`collector`), OpenTelemetry Collector (`otel-collector`) и Tempo (`tempo`). SDK и пакетный экспорт выполняются внутри API и входят в его расход. Приложение закреплено за CPU 0, инфраструктура — CPU 1, k6 — CPU 2, средства наблюдаемости — CPU 3. Лимиты и идентификаторы образов сохраняются в manifest и experiment JSON.

Для каждой пары профиль/повторение рассчитывается разница с baseline. Сравнение включает RPS и ошибки: закрытая модель пользователей обрабатывает меньше запросов при замедлении. Один повтор описывает данный стенд; для оценки разброса доступны дополнительные повторения. Скачки времени, пропуски ресурсов, OOM, рестарты и нарушение полноты трасс делают опыт неуспешным.

## Запуск

Нужны Docker Engine с Compose, минимум четыре CPU, Go, Python и Node.js. Выделенный стенд и ограничения Colima описаны в [ЛР №4](../lab4/README.md).

```sh
make lab6-check
make lab6-smoke
make lab6
```

Полная матрица содержит 20 опытов и занимает около 100 минут со сборкой и подготовкой. `--repetitions N` у `tools/lab6/run.py` задаёт число повторений. Smoke проверяет работу контура и отмечается в Allure как `skipped`.

Артефакты: `build/lab6/runs/<run-id>/`, указатель последнего запуска — `build/lab6/latest.txt`. В каждом опыте сохраняются исходные запросы k6, ресурсы Docker, OTLP JSONL, производный `traces.jsonl` для Chrome Trace, счётчики экспорта SDK и Collector, подтверждение доставки в Tempo и логи. Общий отчёт содержит Markdown, JSON, графики и Allure.

```sh
python3 -m http.server 8088 --directory "$(cat build/lab6/latest.txt)/allure-report"
```

Открыть <http://localhost:8088>. В CI автоматически выполняются unit/race-тесты, короткий `soak-0` во всех четырёх режимах и проверка Grafana. Полная матрица запускается вручную через workflow dispatch с `full_matrix=true`; артефакты сохраняются через `if: always()`.

## Grafana

```sh
make lab6-grafana
make lab6-grafana-down
```

Обзор доступен на <http://localhost:3036/d/lab6-observability>. Он сравнивает режимы по CPU, памяти, RPS и ошибкам; нажатие на столбец открывает выбранный опыт. Prometheus импортирует исторические метрики, Tempo — сохранённые трассы с исходными идентификаторами. [Инструкция](grafana.md) описывает фильтры и просмотр HTTP/SQL spans.

Ресурсы Grafana и Prometheus при просмотре измеряются отдельно. Tempo во время нагрузки учтён непосредственно в новой матрице.

[Результаты](results.md), [выводы](analysis.md), [измерения JSON](measurements.json), [требования](requirements.md).

По просьбе пользователя длительная локальная серия остановлена после 11 завершённых опытов. Текущие численные результаты помечены как неполная серия; smoke всех четырёх режимов завершён.
