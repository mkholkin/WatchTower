# Контекст проекта WatchTower
> Назначение файла: дать новому агенту достаточный контекст для навигации, анализа и безопасного изменения проекта без предварительного полного обхода репозитория. Перед нетривиальной задачей прочитайте файл целиком, а затем перепроверьте затрагиваемые участки по исходникам.

## Правила использования контекста

1. Исходный код — главный источник фактов. Этот файл описывает срез на 20 сентября 2026 года и может устареть после изменений.
2. Для HTTP-контракта доверяйте `api/openapi.yaml`, затем сгенерированному `internal/api/http/v1/gen/openapi.gen.go`, обработчикам и клиенту UI.
3. Для фактической PostgreSQL-схемы доверяйте `migrations/`. Файл `internal/infra/repository/postgres/schema/schema.sql` нужен sqlc и уже расходится с миграциями.
4. Не редактируйте вручную `internal/api/http/v1/gen`, `internal/infra/repository/postgres/sqlcgen` и `internal/service/testmocks`. Меняйте источник и запускайте генератор.
5. Не считайте README, PlantUML, Allure, coverage и лабораторные отчёты доказательством текущего поведения без сверки с кодом.
6. В рабочем дереве могут находиться пользовательские и лабораторные артефакты: `build/`, `tools/`, `tiopo/`, `allure-results/`, `.DS_Store`, `LAB1_RUN=shuffle-2026`. Не удаляйте и не перезаписывайте их без отдельного запроса.
7. Не публикуйте значения JWT secret, паролей БД, Telegram bot token и других секретов из YAML или сущностей контактов.
8. При изменении сквозной функции проверяйте обе реализации хранения — PostgreSQL и MongoDB, — а также Redis-декоратор, API, CLI и UI. CLI сейчас устарел, но остаётся частью дерева.

## Короткая ментальная модель

- `Target` — общая физическая цель проверки: endpoint, протокол и сетевой запрос.
- `Monitor` — пользовательская интерпретация Target: ожидания, частота, статус, контакты и окна обслуживания.
- Scheduler планирует один сетевой запрос на Target. Analyzer применяет один сырой результат к каждому подходящему Monitor.
- PostgreSQL служит каналом между WorkerPool и Analyzer: первый записывает `probe_result`, второй опрашивает необработанные записи.
- Watermill GoChannel передаёт только события изменения конфигурации Target и статуса Monitor. События непостоянны и теряются при остановке процесса.
- Redis — кэш последних `ProbeSummary`, а не брокер и не основное хранилище.
- HTTP API и три фоновые службы живут в одном backend-процессе. React SPA — отдельный процесс.


Срез локального дерева на 20 сентября 2026 года. Карта составлена по исходникам, конфигурации, миграциям и результатам локальной проверки сборки. [Визуальная версия](docs/project-map.html).

## Назначение и архитектура

WatchTower — система самостоятельного размещения для мониторинга доступности ресурсов. Пользователь создаёт мониторы, задаёт ожидания к ответу, подключает контакты и окна обслуживания, смотрит проверки, историю статусов и SLA.

По текущей сборке это **модульный Go-монолит с React SPA**. HTTP API, планировщик, анализатор и уведомления работают в одном backend-процессе. Watermill GoChannel передаёт события в памяти (`Persistent: false`); Redis используется для кэша сводок. Отдельного брокера сообщений в Compose нет.

Основной стек: Go 1.26, Gin, oapi-codegen, pgx/sqlc, Goose, Watermill, Redis; React 19, TypeScript, React Router, Axios, Tailwind, Recharts, react-scripts. Версии здесь взяты из [go.mod](go.mod) и [package.json](watchtower-ui/package.json), а не из установленного окружения.

Объём дерева: 131 Go-файл, 13 Go-файлов тестов с 40 функциями `Test*`, 43 TypeScript/TSX-файла (около 2 900 строк вместе с CSS), 28 HTTP-операций OpenAPI, из которых 26 имеют явный `operationId`, 46 именованных SQL-запросов, 4 миграции и 10 PlantUML-схем. В подсчёт Go входят сгенерированные файлы; Allure, coverage и бинарные артефакты исключены из смыслового разбора.

## Точки входа и слои

- [cmd/api/main.go](cmd/api/main.go) — загрузка YAML, сборка App, Gin, CORS, JWT middleware, HTTP API на `:8080`, `/swagger` и `/openapi.yaml`.
- [internal/app/app.go](internal/app/app.go) — основной узел сборки зависимостей. Запускает PostgreSQL-миграции, создаёт репозитории, Redis, шину событий, сервисы; `Start` запускает три фоновых контура.
- [cmd/cli/main.go](cmd/cli/main.go) — интерактивный CLI с собственной сборкой зависимостей PostgreSQL/Redis; использует те же бизнес-сервисы, не обращается к HTTP API.
- [internal/api/http/v1/handler](internal/api/http/v1/handler) — транспортные обработчики, преобразование запросов и ответов, ошибки, получение пользователя из JWT.
- [internal/service](internal/service) — прикладные сценарии и фоновые службы.
- [internal/domain/entity](internal/domain/entity) — сущности, конфигурации, валидация, доменное поведение.
- [internal/domain/repo](internal/domain/repo) — интерфейсы доступа к данным.
- [internal/infra](internal/infra) — PostgreSQL, MongoDB, Redis, HTTP-пробы и Telegram.
- [pkg/mapper](pkg/mapper) — общие преобразования и ошибки маппинга.

Сервисы зависят от доменных сущностей и интерфейсов репозиториев. Конкретные адаптеры выбираются при сборке приложения. Транспорт вызывает сервисы; UI вызывает транспорт по HTTP/JSON.

## Каталог исходного кода

### Точки входа и конфигурация

- [cmd/api/main.go](cmd/api/main.go) собирает HTTP-сервер: CORS, Swagger, строгий обработчик из OpenAPI, JWT middleware и запуск фоновых служб.
- [cmd/cli/main.go](cmd/cli/main.go) содержит интерактивное меню и прямую сборку сервисов. Это большой самостоятельный адаптер на 1 827 строк; сейчас он отстал от интерфейсов сервисов и не компилируется.
- [configs/types.go](configs/types.go) описывает логирование, JWT, Redis и отдельное подключение к БД для каждого сервиса. [loader.go](configs/loader.go) читает YAML или переменные `WT_*`; [logger.go](configs/logger.go) настраивает `slog` в text/JSON и stdout/stderr/файл.
- [internal/app/app.go](internal/app/app.go) — composition root: миграции, клиенты, репозитории, кэш, сервисы, Watermill и завершение процесса.

### Домен

- [entity/target](internal/domain/entity/target) — общий сетевой Target, его стабильный SHA-256-хеш и полиморфные HTTP/TCP/ICMP-конфигурации.
- [entity/monitor](internal/domain/entity/monitor) — пользовательский Monitor, статусы, ожидания ответа, включение, выключение и проверка активного окна обслуживания.
- [entity/probe](internal/domain/entity/probe) — сырой `Result` и вычисленная для монитора `Summary`; статусы обработки: new, processed, canceled.
- [entity/alert_contact](internal/domain/entity/alert_contact) — контакт, Telegram-конфигурация и частичное обновление.
- [entity/maintenance](internal/domain/entity/maintenance) — разовые и ручные окна, активность и частичное обновление.
- [entity/user](internal/domain/entity/user) — логин и bcrypt-хеш пароля.
- [domain/repo](internal/domain/repo) — восемь портов хранения: user, target, monitor, probe result/summary, contact, maintenance и общие ошибки.

### Прикладные сервисы

- [auth](internal/service/auth) — регистрация, bcrypt, выпуск и разбор JWT, перенос логина через context.
- [monitoring_management](internal/service/monitoring_management) — главный CRUD и оркестрация Target/Monitor; `dto` принимает полиморфные данные, `mappers` строит доменные HTTP/TCP/ICMP-объекты.
- [contacts](internal/service/contacts) и [maintenance](internal/service/maintenance) — CRUD, активация и связи с мониторами.
- [healthcheck](internal/service/healthcheck) — registry проверяющих адаптеров, Scheduler, очередь Target и WorkerPool.
- [analyze](internal/service/analyze) — load shedding, сопоставление результатов с мониторами, evaluator, обновление статусов и событие смены статуса.
- [notification](internal/service/notification) — registry каналов, подписка на события и отдельная очередь отправки.
- [metrics](internal/service/metrics) — сводки, история интервалов и SLA; [common](internal/service/common) — авторизованный пользователь и проверка владения монитором.
- [testmocks](internal/service/testmocks) — сгенерированные mockgen-реализации; [internal/testutil](internal/testutil) — тестовый `slog`.

### Транспорт и инфраструктура

- [api/http/v1/handler](internal/api/http/v1/handler) — семь файлов обработчиков: auth, monitors, contacts, maintenance, metrics, middleware и общая сборка. [error.go](internal/api/http/v1/error.go) переводит сервисные ошибки в HTTP-коды.
- [api/http/v1/gen/openapi.gen.go](internal/api/http/v1/gen/openapi.gen.go) — сгенерированные модели, роуты и строгие интерфейсы; вручную его менять не следует.
- [infra/repository/postgres](internal/infra/repository/postgres) — pgx/sqlc-адаптеры, JSON-маппинг полиморфных типов, SQL-запросы и интеграционные тесты. `sqlcgen` генерируется из `queries` и отдельной `schema`.
- [infra/repository/mongodb](internal/infra/repository/mongodb) — параллельные MongoDB-адаптеры и агрегации метрик.
- [infra/repository/redis](internal/infra/repository/redis) — read-through/write-if-alive кэш последних сводок поверх основного репозитория.
- [infra/probe](internal/infra/probe) — рабочие HTTP prober/evaluator и незавершённые TCP/ICMP-заготовки; [infra/notification](internal/infra/notification) — вызов Telegram Bot API.
- [pkg/mapper](pkg/mapper) — маленький generic-registry преобразователей по ключу.

### Frontend

- [src/App.tsx](watchtower-ui/src/App.tsx), [context/AuthContext.tsx](watchtower-ui/src/context/AuthContext.tsx) и [api](watchtower-ui/src/api) задают маршрутизацию, сессию, Axios interceptors и все HTTP-вызовы.
- [src/pages](watchtower-ui/src/pages) содержит Dashboard, карточку монитора, контакты, окна обслуживания, login/register и 404.
- [src/components/monitors](watchtower-ui/src/components/monitors) — форма, карточка, график задержки, SLA, история, таблица проверок и селекторы связей.
- [src/components/alertContacts](watchtower-ui/src/components/alertContacts) и [maintenanceWindows](watchtower-ui/src/components/maintenanceWindows) — карточки и формы; [shared](watchtower-ui/src/components/shared) и [layout](watchtower-ui/src/components/layout) — переиспользуемый UI и каркас.
- [hooks/useApi.ts](watchtower-ui/src/hooks/useApi.ts) унифицирует загрузку/ошибку/refetch, [types/index.ts](watchtower-ui/src/types/index.ts) вручную дублирует публичные типы OpenAPI, [index.css](watchtower-ui/src/index.css) задаёт Tailwind-слои и тему.

### Контракты, данные и поставка

- [api/openapi.yaml](api/openapi.yaml) — источник HTTP-контракта; копия [watchtower-ui/openapi.yaml](watchtower-ui/openapi.yaml) используется как frontend-артефакт и может рассинхронизироваться.
- [migrations](migrations) создают схему, триггер истории, сервисные роли и временные партиции; [postgres/schema](internal/infra/repository/postgres/schema) — отдельный вход sqlc, поэтому его нужно синхронизировать с миграциями.
- [Dockerfile.backend](Dockerfile.backend), [Dockerfile.postgres](Dockerfile.postgres), [Dockerfile.ui](Dockerfile.ui) и [docker-compose.yaml](docker-compose.yaml) собирают API, PostgreSQL с pg_partman/pg_cron, dev-server UI и Redis.
- [docs/puml](docs/puml), [db.dbml](docs/db.dbml) и изображения в `img/` — проектные схемы; `tools/`, `tiopo/`, `build/` и каталоги `allure-results` — незакоммиченные лабораторные и отчётные артефакты, а не runtime-код.

## Бизнес-модули

- [auth](internal/service/auth/service.go): регистрация, bcrypt, вход и JWT; [common/provider](internal/service/common/provider/user_provider.go) получает пользователя из контекста, [common/ownership](internal/service/common/ownership/monitor.go) содержит общую проверку владения монитором.
- [monitoring_management](internal/service/monitoring_management/service.go): CRUD мониторов, включение/отключение, связи с контактами, поиск или создание Target, пересчёт его интервала, события `target.*`.
- [contacts](internal/service/contacts/service.go): пользовательские контакты оповещений; подключённый канал — Telegram.
- [maintenance](internal/service/maintenance/service.go): окна обслуживания и связи с мониторами; реализованы разовое окно по времени и ручной режим.
- [healthcheck](internal/service/healthcheck/service.go): Scheduler и WorkerPool, выполнение сетевых проверок, сохранение сырых результатов.
- [analyze](internal/service/analyze/service.go): выборка результатов, отбрасывание избыточных старых результатов, оценка по ожиданиям каждого монитора, статусы и сводки.
- [notification](internal/service/notification/service.go): подписка на смену статуса, выбор активных контактов, очередь и пул отправителей.
- [metrics](internal/service/metrics/service.go): проверки за период, последние сводки, история статусов и SLA с проверкой доступа.

## Основной поток

1. UI или CLI вызывает `monitoring_management`. Сервис ищет Target по SHA-256 от endpoint, протокола и JSON сетевой конфигурации. Ожидания монитора, владелец и интервал в хеш не входят.
2. Монитор сохраняет пользовательские ожидания и ссылку на Target. Изменения целей передаются через `target.created`, `target.updated`, `target.deleted`; Scheduler подписан на эти темы.
3. [Scheduler](internal/service/healthcheck/scheduler.go) загружает активные цели из БД, хранит расписание в памяти и примерно раз в секунду отправляет подошедшие цели в Go-канал задач.
4. [WorkerPool](internal/service/healthcheck/worker.go) выбирает prober, выполняет запрос и сохраняет `ProbeResult`: задержку, код ответа, сетевую ошибку и время. Он не определяет UP/DOWN.
5. Анализатор сразу и затем каждые 500 мс читает необработанные результаты **из БД**, находит мониторы, которым нужна оценка, и применяет [HTTP evaluator](internal/infra/probe/http.go). Окно обслуживания имеет приоритет и даёт `maintenance`.
6. Анализатор формирует `ProbeSummary`, обновляет монитор и состояние обработки результата. При смене статуса публикует `monitor.status_changed`. В коде публикация происходит во время оценки, до записей в `commit`; эти операции не образуют единую транзакцию.
7. Notification загружает монитор и отправляет сообщение активным контактам через [Telegram provider](internal/infra/notification/telegram.go). Метрики читаются через отдельный сервис и HTTP API.

В PostgreSQL-ветке App задаёт 200 работников проверок и очередь на 10 000 задач; в MongoDB-ветке — 5 и 100. Анализатор получает `-1` в конфигурации, но конструктор заменяет неположительные значения на лимиты 1000/500. Эти параметры сейчас заданы в коде.

## Модель данных

Главное разделение: **Target — что и как проверять в сети; Monitor — как конкретный пользователь оценивает эту проверку.** Несколько мониторов, включая мониторы разных владельцев, могут ссылаться на одну цель.

- `User → Monitor`: один пользователь владеет многими мониторами.
- `Target → Monitor`: одна цель используется многими мониторами; `Target → ProbeResult` — история сетевых проверок.
- `Monitor → MonitorStatusLog`: интервалы статусов для истории и SLA.
- `Monitor ↔ AlertContact`: связь многие-ко-многим через `monitor_alert_contact`; контакты принадлежат пользователям.
- `Monitor ↔ MaintenanceWindow`: связь многие-ко-многим через `maintenance_window_monitor`; окна также принадлежат пользователям.
- `ProbeSummary`: результат проверки в контексте монитора. В PostgreSQL отдельной таблицы сводок нет: SQL соединяет `probe_result`, `monitor` и `monitor_status_log`; `Create/BulkCreate` PostgreSQL-репозитория намеренно ничего не записывают. MongoDB-адаптер сохраняет отдельные документы сводок.

Источники: [начальная миграция](migrations/20260406140415_init_db.sql), [запросы сводок](internal/infra/repository/postgres/queries/probe_summary.sql), [PostgreSQL-репозиторий сводок](internal/infra/repository/postgres/probe_summary.go), [MongoDB-репозиторий сводок](internal/infra/repository/mongodb/probe_summary.go).

## Хранилища

- **PostgreSQL 18** — основной Compose-сценарий. Таблицы конфигурации, сырые проверки и интервалы статусов. [Триггер](migrations/20260417180047_create_monitor_status_log_trigger.sql) ведёт историю при изменении статуса монитора. `probe_result` разбит по времени; [pg_partman и pg_cron](migrations/20260510112042_create_partition.sql) создают часовые партиции и обслуживают их.
- **Redis 7** — [декоратор ProbeSummaryRepository](internal/infra/repository/redis/probe_summary_repository.go), до 100 последних сводок на монитор по умолчанию, TTL 30 минут, продлеваемый чтением. Запросы за период идут в основное хранилище.
- **MongoDB** — [альтернативные адаптеры](internal/infra/repository/mongodb), выбираемые через `database.<service>.type: mongodb` в App. В Compose MongoDB нет. `InitApp` безусловно выполняет PostgreSQL-миграции, поэтому это пока не автономный запуск только на MongoDB.

SLA в [PostgreSQL-запросе](internal/infra/repository/postgres/queries/metrics.sql) = `UP / (UP + DOWN) × 100` по длительностям интервалов. `MAINTENANCE` и `UNKNOWN` не входят в знаменатель.

## UI и API

Маршруты определены в [App.tsx](watchtower-ui/src/App.tsx): `/login`, `/register`, `/`, `/monitors/:id`, `/alert-contacts`, `/maintenance-windows`, fallback 404. Закрытые страницы обёрнуты в `ProtectedRoute` и `AppLayout`.

- [src/pages](watchtower-ui/src/pages) — страницы; [src/components](watchtower-ui/src/components) — формы, карточки, таблица проверок, график задержки, SLA и лента статусов.
- [AuthContext](watchtower-ui/src/context/AuthContext.tsx) — состояние авторизации; [api/axios.ts](watchtower-ui/src/api/axios.ts) добавляет Bearer-токен из localStorage и перенаправляет на вход при 401.
- [src/api](watchtower-ui/src/api) — клиенты auth, monitors, contacts, maintenance и metrics; [useApi](watchtower-ui/src/hooks/useApi.ts) — загрузка, ошибка и повторный запрос; [src/types](watchtower-ui/src/types/index.ts) — типы UI.

HTTP-контракт: [api/openapi.yaml](api/openapi.yaml). Семейства маршрутов:

- `/auth/register`, `/auth/login`.
- `/monitors`, `/monitors/{monitorId}`, действия `/enable`, `/disable`, связи `/alert-contacts`.
- `/monitors/{monitorId}/checks`, `/sla`, `/status-history`.
- `/alert-contacts`, `/alert-contacts/{contactId}`, `/enable`, `/disable`.
- `/maintenance-windows`, `/maintenance-windows/{windowId}`, связи `/monitors/`.

Каталог `internal/api/http/v1` не означает URL-префикс `/v1`: регистрация роутов в `cmd/api` происходит на корневом роутере. В текущем UI используется HTTP/Axios; WSS, показанный в старой C4-схеме, не следует считать реализованным транспортом.

## Запуск, генерация и тесты

Команды запуска ниже взяты из исходников и Makefile. Для проверки карты отдельно выполнены сборка API и ограниченный прогон тестов без PostgreSQL/Testcontainers.

```sh
# Полный стек: UI :3000, API :8080, PostgreSQL :5432, Redis :6379
docker compose up --build

# API локально, с доступной БД, расширениями и Redis из конфигурации
go run ./cmd/api configs/config.yaml

# CLI: make build собирает именно cmd/cli
make build
./build/watchtower configs/config.yaml

# Frontend отдельно
cd watchtower-ui
npm install --legacy-peer-deps
npm start
```

Конфигурация: [configs/types.go](configs/types.go), [configs/loader.go](configs/loader.go), [локальный YAML](configs/config.yaml), [Docker YAML](docker/config.yaml). Контейнер API получает конфигурацию из `/etc/watchtower/config.yaml`, смонтированную из `docker/`.

- `make gen-api`: [api/oapi-codegen.yaml](api/oapi-codegen.yaml) + OpenAPI → [openapi.gen.go](internal/api/http/v1/gen/openapi.gen.go).
- `make gen-repo`: [sqlc.yaml](sqlc.yaml) + [queries](internal/infra/repository/postgres/queries) и [schema](internal/infra/repository/postgres/schema/schema.sql) → [sqlcgen](internal/infra/repository/postgres/sqlcgen). Миграции и schema для генерации — разные источники, их нужно согласовывать.
- `make mocks`: интерфейсы → [testmocks](internal/service/testmocks); при отсутствии mockgen цель устанавливает инструмент.
- `make test`: генерация моков, затем `go test ./...`. Тесты бизнес-сервисов используют моки и управляемые часы. PostgreSQL-тесты используют Testcontainers с `postgres:15-alpine`; в текущем `setup_test.go` нет build tag `integration`, поэтому общий прогон может требовать Docker. Совместимость тестового образа с расширениями миграций требует отдельной проверки.

Фактическая проверка 20 сентября 2026 года:

- `go build -o /tmp/watchtower-api-check ./cmd/api` — успешно.
- `go build ./...` — неуспешно: `cmd/cli` вызывает удалённые или изменённые методы `GetSummaries`, `ParseToken`, `CreateMonitor` и поле `TotalDowntime`.
- Ограниченный `go test` без PostgreSQL-пакета — неуспешно: healthcheck-моки имеют старую сигнатуру `Probe`, тест monitoring_management ожидает старый результат `CreateMonitor`, один contacts-тест не совпадает по структуре ошибки, maintenance-тест ожидает другой вызов репозитория. При этом тесты analyze, auth и common/provider прошли.
- TypeScript не проверен: `watchtower-ui/node_modules` отсутствует, а установка зависимостей не выполнялась.

## Где менять конкретную функциональность

- Новый HTTP endpoint: OpenAPI → генерация → handler → service; затем UI API-клиент, типы и страница.
- Правила доступности: `domain/entity/monitor/expectations.go`, `infra/probe/http.go`, `service/analyze/service.go`.
- Планирование и частота: `service/healthcheck/scheduler.go`, `monitoring_management/service.go`, `app/app.go`.
- Новый протокол: `domain/entity/target`, `domain/entity/monitor`, DTO/мапперы, prober, evaluator и регистрация в App/CLI.
- Новый канал уведомлений: `domain/entity/alert_contact`, `service/notification`, `infra/notification`, регистрация провайдера и формы UI.
- SLA/история: SQL `queries/metrics.sql`, журнал статусов и его триггер, `service/metrics`, UI-компоненты монитора.
- Изменение данных: миграции, schema для sqlc, queries, генерация, PostgreSQL/MongoDB адаптеры и доменные интерфейсы.

## Границы текущей реализации и документации

1. HTTP-prober зарегистрирован; [TCP](internal/infra/probe/tcp.go) и [ICMP](internal/infra/probe/icmp.go) содержат `panic("implement me")`. `Validate()` у HTTP/TCP/ICMP expectations и у ICMP network config тоже не реализованы. Эти протоколы и доменная валидация не готовы, даже если типы и мапперы уже существуют.
2. Основной API собирается, но весь Go-модуль и цель `make build` сейчас ломаются на устаревшем CLI. Несколько unit-тестов также не соответствуют текущим интерфейсам; сохранённые Allure-отчёты не доказывают состояние текущего дерева.
3. README и C4 упоминают Email/SMTP; App регистрирует только Telegram. В notification нет отдельного фильтра переходов в/из maintenance: при событии уведомляются активные контакты. Обещание README «mute alerts» нельзя трактовать как полное подавление таких уведомлений.
4. Операции связи требуют отдельного аудита авторизации: maintenance-сервис при добавлении/удалении связи загружает объекты, но не сравнивает их владельца с текущим пользователем; monitoring_management проверяет владельца монитора, но не владельца подключаемого контакта.
5. Обработка ошибок непоследовательна: `service.Error.Error()` возвращает результат пустого `errors.Join()`, `getOrCreateTarget` проглатывает не-`NotFound` ошибку репозитория, `EnableMonitor` проглатывает ошибку синхронизации Target. Это влияет на диагностику и HTTP-ответы.
6. Compose задаёт `REACT_APP_API_BASE_URL`, но Axios читает `REACT_APP_API_URL`. Локальный fallback `http://localhost:8080` совпадает с Compose-портом, однако указанная в Compose переменная не управляет клиентом.
7. MongoDB-адаптеры существуют, но Compose MongoDB не поднимает, а `InitApp` безусловно выполняет PostgreSQL-миграции. CLI также собирает PostgreSQL-репозитории напрямую.
8. PostgreSQL-миграции и `postgres/schema/schema.sql` расходятся: миграции задают партиционированный составной ключ `probe_result`, дополнительные `UNIQUE`, а schema для sqlc — обычный UUID primary key без этих ограничений.
9. Цели benchmark ссылаются на отсутствующий каталог `benchmarks/`. [tiopo/lab1/README.md](tiopo/lab1/README.md) и [results.md](tiopo/lab1/results.md) упоминают отсутствующие цели и скрипты.
10. [docs/puml](docs/puml) содержит C4 L1–L4, ER, use-case, BPMN и sequence-диаграммы; [db.dbml](docs/db.dbml) — модель БД. Это полезные исходные материалы, но подключённые возможности следует сверять с App, адаптерами и миграциями.

## Порядок чтения для знакомства

`cmd/api/main.go` → `internal/app/app.go` → `domain/entity/target` и `domain/entity/monitor` → `monitoring_management/service.go` → `healthcheck/{scheduler,worker}.go` → `analyze/service.go` → SQL сводок/метрик → `watchtower-ui/src/App.tsx` и `MonitorDetailPage.tsx`.

## Полный каталог HTTP API

Все пути регистрируются от корня, без префикса `/v1`. Кроме регистрации и входа, операции требуют Bearer JWT; middleware пропускает отсутствующий или неверный токен дальше, после чего `UserProvider` возвращает unauthorized.

| Метод | Путь | Назначение |
|---|---|---|
| POST | `/auth/register` | Регистрация пользователя |
| POST | `/auth/login` | Вход и получение JWT |
| GET | `/monitors` | Список мониторов пользователя |
| POST | `/monitors` | Создание монитора |
| GET | `/monitors/{monitorId}` | Детали монитора |
| PATCH | `/monitors/{monitorId}` | Частичное обновление |
| DELETE | `/monitors/{monitorId}` | Удаление монитора |
| GET | `/monitors/{monitorId}/checks` | Последние проверки или проверки за период |
| GET | `/monitors/{monitorId}/sla` | SLA за период |
| GET | `/monitors/{monitorId}/status-history` | История интервалов статуса |
| POST | `/monitors/{monitorId}/enable` | Включение монитора |
| POST | `/monitors/{monitorId}/disable` | Отключение монитора |
| POST | `/monitors/{monitorId}/alert-contacts` | Привязка контакта |
| DELETE | `/monitors/{monitorId}/alert-contacts` | Отвязка контакта |
| GET | `/alert-contacts` | Список контактов пользователя |
| POST | `/alert-contacts` | Создание Telegram-контакта |
| GET | `/alert-contacts/{contactId}` | Детали контакта |
| PATCH | `/alert-contacts/{contactId}` | Изменение контакта |
| DELETE | `/alert-contacts/{contactId}` | Удаление контакта |
| POST | `/alert-contacts/{contactId}/enable` | Включение контакта |
| POST | `/alert-contacts/{contactId}/disable` | Отключение контакта |
| GET | `/maintenance-windows` | Список окон обслуживания |
| POST | `/maintenance-windows` | Создание окна |
| GET | `/maintenance-windows/{windowId}` | Детали окна |
| PATCH | `/maintenance-windows/{windowId}` | Изменение окна |
| DELETE | `/maintenance-windows/{windowId}` | Удаление окна |
| POST | `/maintenance-windows/{windowId}/monitors/` | Привязка монитора к окну; завершающий `/` входит в контракт |
| DELETE | `/maintenance-windows/{windowId}/monitors/` | Отвязка монитора от окна |

Swagger доступен по `/swagger`, исходный контракт — по `/openapi.yaml`.

## События и асинхронные контуры

| Topic | Publisher | Subscriber | Payload |
|---|---|---|---|
| `target.created` | monitoring_management | Scheduler | `TargetEvent{ID}` |
| `target.updated` | monitoring_management | Scheduler | `TargetEvent{ID}` |
| `target.deleted` | monitoring_management | Scheduler | `TargetEvent{ID}`; сервис почти всегда публикует updated после disable, а не deleted |
| `monitor.status_changed` | Analyzer | Notification | monitor ID, старый и новый статус, время |

События не образуют транзакцию с БД. Analyzer публикует смену статуса до `commit`; сбой записи после публикации способен оставить уведомление без соответствующего сохранённого состояния.

## Ключевые runtime-параметры

- Scheduler: минимальный интервал 1 секунда, tick — 1 секунда + 10 мс.
- HTTP client prober: timeout 30 секунд.
- PostgreSQL healthcheck: 200 workers, очередь 10 000 Target.
- MongoDB healthcheck: 5 workers, очередь 100 Target.
- Analyzer: опрос каждые 500 мс; fetch limit 1 000; при перегрузке оставляет 500 новых результатов, старые помечает canceled.
- Notification: 30 workers, очередь 256; при заполнении задача отбрасывается.
- Redis: максимум 100 последних сводок на Monitor, TTL 30 минут. TTL продлевается чтением; запись обновляет только уже существующую живую очередь.
- PostgreSQL `probe_result`: часовые партиции, три будущие партиции, обслуживание pg_partman по cron каждый час.

## Зависимости между слоями

Номинальное направление: `cmd/api` и `handler` → `service` → `domain/repo` и `domain/entity` → `infra` как реализация портов. `internal/app` связывает реализации. Архитектура не полностью чистая: доменные validation errors импортируют пакет `internal/service`, а infra evaluator и prober реализуют интерфейсы сервисного слоя.

App выбирает PostgreSQL или MongoDB отдельно для monitoring, maintenance, auth, metrics, healthchecker, analyzer и contacts. Для PostgreSQL создаётся отдельный pool на сервис; для MongoDB клиент переиспользуется по DSN. Notification всегда использует хранилище monitoring. Миграции выполняются безусловно через отдельный `database.migrations.dsn`, поэтому полностью MongoDB-запуск не поддержан.

## Точные источники для типовых изменений

- HTTP endpoint: `api/openapi.yaml` → `make gen-api` → `internal/api/http/v1/handler` → сервис → UI-клиент и типы.
- PostgreSQL-поле или таблица: новая миграция → синхронизация `postgres/schema/schema.sql` → SQL в `queries` → `make gen-repo` → mapper/repository.
- Новый протокол: target config → monitor expectations → DTO/mappers → prober → evaluator → registry в App → OpenAPI → UI. Одних доменных типов недостаточно.
- Новый канал уведомлений: contact config → repository mapping → provider → registry → OpenAPI/UI.
- Новая метрика: `MetricQueryService` → `AnalyticsRepository` для обеих БД → handler/OpenAPI → UI.
- Изменение авторизации: auth service, strict middleware, `UserProvider`, ownership helpers и все операции связей.
- Изменение фоновой конкуренции: `internal/app/app.go` и конфигурационные структуры сервисов; большинство значений пока зашито в код.

## Дополнительные известные риски

- `maintainTargetConsistency` заявляет учёт активных мониторов, но PostgreSQL `GetAllMonitorsByTargetID` не фильтрует `is_active`; отключённые мониторы могут удерживать Target активным и влиять на минимальный интервал.
- `getOrCreateTarget` при ошибке, отличной от not found, возвращает `tgt, nil`, скрывая сбой репозитория.
- `EnableMonitor` возвращает `nil`, если синхронизация Target завершилась ошибкой.
- `service.Error.Error()` вызывает пустой `errors.Join()` и не формирует полезное сообщение.
- Ownership проверяется не во всех операциях связей. Нельзя считать идентификатор связанного контакта или окна безопасным только потому, что Monitor принадлежит пользователю.
- CORS разрешает все origins. Это осознанная dev-конфигурация, а не готовая production-политика.
- Target hash включает endpoint, protocol и JSON network config, но не пользователя, ожидания и interval. Порядок сериализации map в Go стабилен для `encoding/json`, однако смена представления конфигурации меняет идентичность Target.
- В PostgreSQL `ProbeSummaryRepository.Create/BulkCreate` — no-op. Сводки восстанавливаются SQL JOIN по сырым результатам и истории статусов; не добавляйте таблицу summary только из предположения о симметрии с MongoDB.
- При заполнении очереди Notification молча не повторяет задачу, а логирует drop.
- UI хранит JWT в localStorage и при 401 очищает его и перенаправляет на `/login`.

## Чек-лист агента перед изменением

1. Найдите точку входа сценария и интерфейс сервиса.
2. Проследите путь до сущностей, портов и обеих реализаций репозитория.
3. Проверьте OpenAPI, handler, UI-клиент и ручные TypeScript-типы.
4. Определите, затрагиваются ли Watermill events, Redis cache или status log trigger.
5. Не редактируйте generated-файлы вручную.
6. Добавьте или обновите тесты; сначала учтите, что часть существующих тестов и mocks устарела.
7. Минимально проверьте `go build -o /tmp/watchtower-api-check ./cmd/api`.
8. Для полного Go-прогона ожидайте текущие ошибки CLI и тестов, пока они не исправлены.
9. Для PostgreSQL integration tests нужен Docker; тестовый образ `postgres:15-alpine` отличается от Compose PostgreSQL 18 с расширениями.
10. Для UI сначала установите зависимости; без `watchtower-ui/node_modules` TypeScript-проверка невозможна.

## Глоссарий

- **Target** — дедуплицированная сетевая цель.
- **Monitor** — пользовательские правила оценки Target.
- **ProbeResult** — сырой результат сетевого запроса.
- **ProbeSummary** — результат оценки ProbeResult для конкретного Monitor.
- **Status log** — интервальная история UP/DOWN/MAINTENANCE/UNKNOWN.
- **MaintenanceWindow** — разовый временной диапазон или ручной переключатель.
- **AlertContact** — активируемый канал доставки; реализован Telegram.
- **Scheduler** — планировщик Target в памяти.
- **WorkerPool** — конкурентное выполнение проверок.
- **Analyzer** — преобразование сырых результатов в пользовательские статусы.
- **Notification** — подписчик на смену статуса и пул отправителей.
