# WebLab № 3 — отчёт о выполнении

Ветка: `WEB-3`.

Реализован REST API v2 по контракту WebLab № 2. OpenAPI остаётся источником истины, а серверные и клиентские типы генерируются из него. Старый API сохранён как `/api/v1`, графический интерфейс продолжает работать через совместимые маршруты, терминальный интерфейс переведён на API v2.

## Соответствие требованиям

| № | Требование лабораторной | Где реализовано |
|---:|---|---|
| 1 | Реализовать REST API по OpenAPI; контракт — источник истины | [api/openapi.yaml](../../api/openapi.yaml), [api/oapi-codegen.yaml](../../api/oapi-codegen.yaml), [Makefile](../../Makefile), сгенерированные типы и маршруты в [internal/api/http/v2/gen/openapi.gen.go](../../internal/api/http/v2/gen/openapi.gen.go), обработчики в [internal/api/http/v2/handler](../../internal/api/http/v2/handler) |
| 2 | Подключить Swagger | Маршруты `/swagger` и `/openapi.yaml` в [cmd/api/main.go](../../cmd/api/main.go), встроенный контракт в [api/spec.go](../../api/spec.go) |
| 3 | Обеспечить работоспособность Swagger UI | Swagger UI 5 загружает OpenAPI 3.1 из `/openapi.yaml`; маршруты проверены в [cmd/api/main_test.go](../../cmd/api/main_test.go) |
| 4 | Чистая архитектура, Repository и три модели | Интерфейсы Repository: [internal/domain/repo](../../internal/domain/repo); модели БД: [internal/infra/repository/postgres/sqlcgen/models.go](../../internal/infra/repository/postgres/sqlcgen/models.go); доменные модели: [internal/domain/entity](../../internal/domain/entity); API DTO: [internal/api/http/v2/gen/openapi.gen.go](../../internal/api/http/v2/gen/openapi.gen.go); слой контроллеров и преобразований: [internal/api/http/v2/handler](../../internal/api/http/v2/handler) |
| 5 | Одновременная работа TUI, GUI и API; TUI использует новый API | API и GUI запускаются через [docker-compose.yaml](../../docker-compose.yaml); существующий GUI сохранён; CLI использует сгенерированный v2-клиент через [cmd/cli/main.go](../../cmd/cli/main.go) и [internal/api/http/v2/client](../../internal/api/http/v2/client) |
| 6 | Сохранить старый API как `/api/v1`, новый разместить в `/api/v2` | Регистрация обоих наборов маршрутов в [cmd/api/main.go](../../cmd/api/main.go); тест наличия версий в [cmd/api/main_test.go](../../cmd/api/main_test.go) |

## Реализованные возможности

- Все 24 операции контракта: авторизация, мониторы, проверки и SLA, Telegram-контакты, окна обслуживания и связи между ресурсами.
- Bearer JWT, единый формат ошибок, корректные статусы, `Location` и `WWW-Authenticate`.
- Частичные обновления через `PATCH`, включая ручное включение окна обслуживания.
- Проверка владения связанными ресурсами; связи сохраняются вместе с монитором в одной транзакции, повторные `PUT` идемпотентны.
- Ограничения контракта проверяются на входе: длины строк, уникальность идентификаторов и неизвестные JSON-поля.
- Сгенерированный API-клиент для CLI; прежний режим прямого подключения CLI по конфигурационному файлу оставлен для совместимости.
- Контракт встроен в бинарный файл API, поэтому Swagger не зависит от текущего каталога процесса.

## Запуск трёх интерфейсов

~~~bash
docker compose up --build
go run ./cmd/cli http://localhost:8080
~~~

После запуска доступны GUI `http://localhost:3000`, Swagger `http://localhost:8080/swagger`, API v1 `http://localhost:8080/api/v1` и API v2 `http://localhost:8080/api/v2`.

## Проверка

| Команда | Результат |
|---|---|
| `GOCACHE=/tmp/watchtower-go-cache make validate-api` | Пройдено: структура OpenAPI, REST-правила и примеры корректны |
| `go test ./internal/api/http/v2/... ./cmd/api ./cmd/cli` | Пройдено: v2-сервер, клиент, маршруты и CLI |
| `go test ./internal/service/...` | Пройдено: регрессионные тесты сервисного слоя |
| `go build ./cmd/api` и `go build ./cmd/cli` | Обе программы собраны |
| `go vet ./cmd/... ./internal/api/http/v2/... ./internal/service/...` | Пройдено |
| `docker compose config --quiet` | Пройдено: конфигурация совместного запуска корректна |
| `go test ./...` | Все обычные тесты пройдены; два существующих интеграционных теста PostgreSQL не запустились из-за заполненного диска Docker VM (`No space left on device`) |
