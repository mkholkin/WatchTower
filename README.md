# WatchTower

## 1. Идея проекта
WatchTower — это self-hosted система для многопользовательского мониторинга доступности веб-ресурсов. Проект реализует архитектуру с дедупликацией целей мониторинга, что позволяет множеству пользователей отслеживать одни и те же ресурсы без кратного увеличения сетевой нагрузки и объема хранимых данных.

## 2. Описание предметной области
Предметная область охватывает автоматизацию контроля состояния распределенных информационных систем и сетевой инфраструктуры. Основной акцент делается на разделении физического опроса ресурсов и логического представления результатов для пользователей.

Ключевыми сущностями предметной области являются: **Users** (владельцы конфигураций), **Targets** (физические ресурсы: URL или IP, подлежащие проверке), **UserMonitors** (персонализированные настройки мониторинга), **CheckResults** (хронологические записи телеметрии), **MaintenanceWindows** (интервалы планового обслуживания, исключаемые из оповещений подсчета sla).

## 3. Анализ аналогичных решений

| Критерий сравнения                                    | UptimeRobot (SaaS)                           | Uptime Kuma (Open Source)                | Gatus (Open Source)                           | **WatchTower (Разрабатываемое)**                              |
|:------------------------------------------------------|:---------------------------------------------|:-----------------------------------------|:----------------------------------------------|:--------------------------------------------------------------|
| **Приватность данных**                                | Низкая (Данные хранятся в облаке провайдера) | Высокая (Self-hosted)                    | Высокая (Self-hosted)                         | **Высокая (Self-hosted)**                                     |
| **Архитектура и производительность**                  | Закрытый код                                 | Node.js (Однопоточная модель, SQLite)    | Go (Высокая производительность, YAML конфиги) | **Go (Высокая производительность, PostgreSQL + Redis)**       |
| **Оптимизация опроса (Дедупликация)**                 | Нет данных (SaaS)                            | Нет (Каждый монитор — отдельный процесс) | Нет (Config-as-Code)                          | **Да (Объединение одинаковых целей от разных пользователей)** |
| **Возможность мониторинга ресурсов в локальной сети** | Нет (Saas)                                   | Да                                       | Да                                            | **Да**                                                        |
| **Многопользовательский режим**                       | Есть (Платный тариф)                         | Есть                                     | Отсутствует                                   | **Есть**                                                      |

## 4. Обоснование целесообразности и актуальности
Разработка WatchTower обусловлена потребностью в инструменте, который сочетает простоту использования SaaS-решений с безопасностью и контролем self-hosted систем, при этом превосходя существующие Open Source аналоги в производительности.

## 5. Краткое описание акторов (ролей)
*   **Пользователь (User):** Основной потребитель системы. Создает и настраивает мониторы (`UserMonitors`), задает периоды обслуживания (`SuspendIntervals`), просматривает дашборды и получает уведомления об инцидентах.
*   **Неавторизованный пользователь (Admin):** Ничего не может. Только авторизоваться.

## 6. Use-case диаграмма
```plantuml
@startuml
left to right direction
skinparam packageStyle rectangle
skinparam actorStyle awesome

actor "Пользователь" as User
actor "Неавторизованный\nПользователь" as UnauthorizedUser

package "Система WatchTower" {
    usecase "Авторизация и регистрация" as UC_Auth
    usecase "Управление каналами оповещения" as UC_Contacts

    usecase "Создание монитора" as UC_CreateMon
    usecase "Настройка частоты опроса" as UC_SetInterval
    usecase "Управление окнами обслуживания" as UC_SetSuspend
    usecase "Включение/Пауза монитора" as UC_Pause

    usecase "Просмотр дашборда\n(Текущие статусы)" as UC_Dashboard
    usecase "Просмотр графиков истории" as UC_History
    usecase "Расчет SLA" as UC_CalcSLA
}

User --> UC_Contacts
User --> UC_CreateMon
User --> UC_SetSuspend
User --> UC_Pause
User --> UC_Dashboard

UnauthorizedUser --> UC_Auth
UC_CreateMon ..> UC_SetInterval : <<include>>
UC_Dashboard ..> UC_History : <<extend>>
UC_History ..> UC_CalcSLA : <<include>>

@enduml
```

## 7. ER-диаграмма
```plantuml
@startchen
!theme plain
left to right direction

entity Пользователь {
    Логин <<key>>
    Хеш пароля
}

entity Монитор {
    ID <<key>>
    Название
    Статус
    Expectations
    Интервал опроса
    Включен ли
}

entity Цель {
    ID <<key>>
    endpoint
    протокол или тип
    конфигурация
}

entity Результат_опроса {
    ID <<key>>
    Ошибка сети флаг
    Status code
    Время задержки latency
    Метаинформация
    Время создания
}

entity Check_Summary {
    ID <<key>>
    Статус
}

entity История_Изменений_Монитора {
    Статус
    Время начала
    Время конца
}

entity Канал_Оповещения {
    ID <<key>>
    Тип
    Конфигурация
}

entity Окно_Обслуживания {
    ID <<key>>
    Название
    Описание
    Тип
    Конфигурация
}

relationship "Владеет" as owns1 {
}

relationship "Владеет" as owns2 {
}

relationship "Ссылается" as refs1 {
}

relationship "Ссылается" as refs2 {
}

relationship "Ссылается" as refs3 {
}

relationship "Ссылается" as refs4 {
}

relationship "Ссылается" as refs5 {
}

relationship "Ссылается" as refs6 {
}

relationship "Оповещает через" as alerts {
}

Пользователь =1= owns1
owns1 =N= Монитор

Пользователь =1= owns2
owns2 =N= Канал_Оповещения

Монитор =N= refs1
refs1 =1= Цель

Результат_опроса =n= refs2
refs2 =1= Цель

Монитор =N= alerts
alerts =N= Канал_Оповещения

История_Изменений_Монитора =N= refs3
refs3 =1= Монитор

Окно_Обслуживания =N= refs4
refs4 =N= Монитор

Check_Summary =1= refs5
Результат_опроса =1= refs5

Check_Summary =N= refs6
Монитор =1= refs6

@endchen
```

## 8. Краткое описание сценариев использования

### Сценарий 1. Добавление монитора для популярного ресурса
> Этот сценарий демонстрирует ключевую архитектурную особенность системы — **дедупликацию целей**.

1.  **Пользователь** инициирует создание нового Монитора, указывая адрес ресурса (URL), желаемую частоту проверок (например, раз в 30 секунд) и конфигурацию.
2.  **Пользователь** выбирает каналы для получения уведомлений (Telegram, Email).
3.  **Система** проверяет, находится ли указанный URL уже под наблюдением (существует ли активная Цель).
4.  **Система** обнаруживает существующую Цель, которая в данный момент опрашивается реже (например, раз в 60 секунд).
5.  **Система** обновляет конфигурацию Цели: устанавливает новую, более высокую частоту опроса (30 секунд), чтобы удовлетворить требования всех подписчиков.
6.  **Система** создает новую подписку (Пользовательский Монитор), связывая Пользователя с данной Целью.
7.  **Система** подтверждает успешное создание монитора.

---

### Сценарий 2. Настройка планового окна обслуживания
1. **Пользователь** инициирует создание нового окна обслуживания.
2. **Пользователь** выбирает тип окна, вводит название и описание, выбирает мониторы, к которым применяется окно.
3. **Система** создает новое окно обслуживания для заданных мониторов.
4. **Система** подтверждает успешное создание окна.

---

### Сценарий 3. Обнаружение сбоя и маршрутизация уведомлений
*   **Предусловия:**
    *   Получено событие об изменении статуса на "DOWN" для ресурса `api.myservice.com`.
    *   На ресурс подписаны два пользователя:
        *   Пользователь A (Окно обслуживания сейчас активно).
        *   Пользователь B (Окон обслуживания нет, мониторинг активен).
        
1.  **Сервис Оповещений** получает сигнал об инциденте.
2.  **Сервис Оповещений** определяет список всех мониторов, связанных с этим ресурсом.
3.  **Сервис Оповещений** начинает итерацию по мониторам для применения фильтров:
4.  *Обработка монитора пользователя A:
    *   Статус монитора определяется, как "MAINTAINING"
    *   **Действие:** Уведомление блокируется (Skipped).
5.  *Обработка монитора пользователя B:*
    *   Статус монитора определяется, как "DOWN"
    *   **Действие:** отправка оповещения в каналы связи связанные с монитором пользователя B.

## 9 Формализация ключевых бизнес-процессов
### Процесс добавления монитора с дедупликацией целей
```plantuml
@startuml
!theme plain
title BPMN: Бизнес-процесс добавления нового монитора

' --- Стилизация под классический BPMN ---
skinparam conditionStyle diamond
skinparam swimlane {
    BorderColor Black
    BorderThickness 1
    TitleBackgroundColor LightGray
    TitleFontColor Black
}
skinparam activity {
    BackgroundColor White
    BorderColor Black
    ArrowColor Black
}

' Объявление параллельных дорожек
|Пользователь|
|Система|

' --- Ход процесса ---

|Пользователь|
start
:Ввод параметров монитора\n(Адрес, Протокол, Интервал,\n параметры спецефичные для протокола);
:Выбор контактов для оповещения;
:Отправка запроса на создание;

|Система|
:Валидация входных параметров;
:Поиск существующей физической Цели (ресурса);

if (Цель уже существует?) then (Да)
    if (Запрошенный интервал\nопроса меньше текущего?) then (Да)
        :Обновление конфигурации Цели\n(Применение минимального интервала);
    else (Нет)
'        :Сохранение текущего интервала Цели;
    endif

else (Нет)
    :Регистрация новой физической Цели\n(Target);
endif

:Создание логического Монитора,\nпривязанного к Цели;

|Пользователь|
:Получение уведомления\nоб успешном создании;
stop

@enduml
```

### Процесс опроса ресурса и обработки результатов
```plantuml
@startuml
!theme plain
title BPMN: Бизнес-процесс проверки ресурса и оповещения

' --- Стилизация под классический BPMN ---
skinparam conditionStyle diamond
skinparam swimlane {
    BorderColor Black
    BorderThickness 1
    TitleBackgroundColor LightGray
    TitleFontColor Black
}
skinparam activity {
    BackgroundColor White
    BorderColor Black
    ArrowColor Black
}

' Объявление параллельных дорожек
|Планировщик (Scheduler)|
|Сервис Анализа (Analyzer)|
|Сервис Оповещений (Notifier)|
|Отслеживаемый ресурс|

' --- Ход процесса ---

|Планировщик (Scheduler)|
start
:Триггер по расписанию;
:Выполнение сетевого запроса;

|Отслеживаемый ресурс|
: Ответ (или таймаут));

|Планировщик (Scheduler)|
: Сохранение результата проверки\nв персистентное хранилище;

|Сервис Анализа (Analyzer)|
:Получение результата;
:Определение текущего состояния\n цели на мониторе\n(UP / DOWN / MAINTENANCE));

if (Статус изменился на DOWN) then (Да)
    |Сервис Оповещений (Notifier)|
    :Отправка уведомлений о сбое\nпо включенным каналам оповещения\nна мониторе;
    stop
else (Нет)
    |Сервис Анализа (Analyzer)|
    stop
endif

@enduml
```
## 10 C4 диаграммы
### Контекстная диаграмма (C4 Level 1)
```plantuml
@startuml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Context.puml

' Настройки отображения
LAYOUT_WITH_LEGEND()
title Диаграмма контекста (C4 Level 1) - Система мониторинга WatchTower

' === Акторы (Пользователи) ===
Person(user, "Пользователь", "DevOps-инженер, системный администратор или владелец сайта. Настраивает проверки и получает алерты.")

' === Наша система ===
System(watchtower, "WatchTower", "Платформа для непрерывного мониторинга доступности сервисов, расчета SLA и маршрутизации уведомлений об инцидентах.")

' === Внешние системы ===
System_Ext(target_systems, "Целевые ресурсы\n(Web, API, Servers)", "Внешние сайты, API и серверы, доступность которых проверяется системой.")
System_Ext(notification_apis, "Платформы оповещений\n(Telegram API)", "Внешние провайдеры связи для доставки сообщений об инцидентах.")

' === Взаимодействия (Связи) ===
Rel(user, watchtower, "Настраивает мониторы, просматривает дашборды и метрики")

Rel(watchtower, target_systems, "Опрашивает статус")
Rel(watchtower, notification_apis, "Отправляет данные об инцидентах")

Rel(notification_apis, user, "Доставляет уведомления")

@enduml
```

### Диаграмма контейнеров (C4 Level 2)
```plantuml
@startuml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Container.puml
LAYOUT_WITH_LEGEND()

title Диаграмма контейнеров (C4 Level 2) - Система мониторинга WatchTower

' === Акторы ===
Person(user, "Пользователь", "DevOps-инженер или владелец ИТ-сервиса. Настраивает мониторинг, правила оповещений и просматривает дашборды.")

' === Границы нашей системы (WatchTower) ===
System_Boundary(watchtower, "WatchTower") {

    Container(spa, "Frontend Application", "SPA (TypeScript, React)", "Предоставляет веб-интерфейс для управления логическими мониторами, настройки контактов, просмотра графиков задержки ответа и SLA.")

    Container(backend, "Backend Application", "Go (Golang)", "Реализует REST API для мониторинга, выполняет непосредственные проверки доступности ресурсов.")

    ContainerDb(db, "БД", "PostgreSQL", "Хранит пользователей, конфигурации, цели, также исторические временные ряды проверок.")

    ContainerDb(redis, "In-Memory Cache", "Redis", "Хранит «горячие» данные: последние известные состояния ресурсов (Last Known State) для мгновенной загрузки дашбордов.")
}

System_Ext(target_systems, "Целевые ресурсы\n(Web, API, Servers)", "Внешние сайты, API и серверы, доступность которых отслеживается системой.")
System_Ext(notification_apis, "Платформы оповещений\n(Telegram API, SMTP)", "Внешние провайдеры связи для доставки сообщений об инцидентах.")


' === Взаимодействия (Связи) ===

' Пользователь -> Frontend
Rel(user, spa, "Просматривает метрики и настраивает конфигурацию", "HTTPS")

' Frontend -> Backend
Rel(spa, backend, "Запрашивает данные и отправляет команды", "JSON/HTTPS/WSS")

' Backend <-> Базы Данных
Rel(backend, db, "Чтение/запись конфигураций и исторических данных", "TCP/SQL")
Rel(backend, redis, "Чтение/обновление текущих статусов", "TCP/RESP")

' Backend -> Внешние системы
Rel(backend, target_systems, "Выполняет проверки доступности", "HTTP/TCP/ICMP")
Rel(backend, notification_apis, "Отправляет сформированные алерты", "HTTPS/SMTP")

' Внешние системы -> Пользователь
'Rel(notification_apis, user, "Доставляет уведомления")

@enduml
```

### Диаграмма компонентов (C4 Level 3)
```plantuml
@startuml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Container.puml
'skinparam linetype polyline
LAYOUT_WITH_LEGEND()

title Диаграмма контейнеров (C4 Level 3) - WatchTower Backend Components

' === External ===
Person(user, "Пользователь", "DevOps/Admin")
System_Ext(target_systems, "Target Resources", "HTTP/TCP/ICMP Targets")
System_Ext(notification_apis, "Notification APIs", "Telegram, SMTP")

System_Boundary(watchtower, "WatchTower System") {
    Container(spa, "Frontend SPA", "React, TS", "Web Dashboard")

    ContainerDb(db, "PostgreSQL", "Relational DB", "Persistent Data")
    ContainerDb(redis, "Redis", "Key-Value Store", "Hot Cache & Last Known State")

    System_Boundary(backend, "Backend Application") {

        ' --- Data Access Layer ---
        Container(repository, "Repository Layer", "Go Interface", "Абстракция доступа к данным (Postgres + Redis)")

        ' --- API Layer ---
        Container(auth_api, "Auth API", "Go/Gin", "REST Controller")
        Container(monitoring_api, "Monitoring API", "Go/Gin", "REST Controller")

        ' --- Domain Services (Business Logic) ---
        Container(auth_service, "Auth Service", "Go", "Logic: Login, JWT")
        Container(monitoring_service, "Monitoring Mgmt Service", "Go", "Logic: CRUD Monitors")
        Container(metric_query_service, "Metric Query Service", "Go", "Logic: Read Stats/History (CQRS Read)")
        Container(maintenance_service, "Maintenance Service", "Go", "Logic: Windows & Schedules")
        Container(contact_service, "Contact Service", "Go", "Logic: Alert Contacts")

        ' --- Background Workers ---
        System_Boundary(workers, "Background Workers") {
            Container(healthchecker, "Health Checker", "Go Routine", "Scheduler & Executor. Writes results to DB.")
            Container(analyzer, "Analyzer", "Go Routine", "Polls results from DB, evaluates logic.")
            Container(notifier, "Notifier", "Go Routine", "Sends alerts via providers.")
            Container(notification_provider, "Notification Provider", "Interface", "Adapter for external APIs")
        }
    }
}

' === Relationships ===

' Frontend to API
Rel(user, spa, "Uses")
Rel(spa, monitoring_api, "JSON/HTTPS")
Rel(spa, auth_api, "JSON/HTTPS")

' API to Services
Rel(auth_api, auth_service, "Calls")
Rel(monitoring_api, monitoring_service, "Calls")
Rel(monitoring_api, metric_query_service, "Calls")
Rel(monitoring_api, maintenance_service, "Calls")
Rel(monitoring_api, contact_service, "Calls")

' Services to Repository
Rel(auth_service, repository, "Read/Write")
Rel(monitoring_service, repository, "Read/Write")
Rel(maintenance_service, repository, "Read/Write")
Rel(contact_service, repository, "Read/Write")
Rel(metric_query_service, repository, "Read Only")

' Workers Interactions
Rel(healthchecker, repository, "Writes Results")
Rel(analyzer, repository, "Reads Results / Writes Status")
Rel(analyzer, notifier, " ")
Rel(notifier, notification_provider, "Uses")
Rel(notifier, repository, "Reads Contact Info")

' Notification Logic
Rel(notification_provider, notification_apis, "Sends")

' Checking Logic
Rel(healthchecker, target_systems, "Pings")

' Persistence
Rel(repository, db, "SQL")
Rel(repository, redis, "RESP")

@enduml
```

### Диаграмма классов (C4 Level 4)
```plantuml
@startuml
skinparam linetype ortho
'left to right direction
namespace healthcheck {
    class "HealthChecker" << (S,Aquamarine) >> {
        + Run(ctx context.Context) error

    }
    class "HealthCheckerConfig" << (S,Aquamarine) >> {
        + WorkerCount int
        + TaskQueueSize int

    }
    interface "Prober"  {
        + Probe(ctx context.Context, target *target.Target) (probe.ProbeResult, error)
    }
    class "ProberRegistry" << (S,Aquamarine) >> {
        + Register(protocol target.Protocol, prober Prober) 
        + Get(protocol target.Protocol) (Prober, error)
    }
    class "Scheduler" << (S,Aquamarine) >> {
        + Run(ctx context.Context) error

    }

    class "WorkerPool" << (S,Aquamarine) >> {
        + Run(ctx context.Context) 

    }
    class "scheduleEntry" << (S,Aquamarine) >> {
        + Target target.Target
    }
}

enum "target.Protocol" {
    HTTP
    TCP
    ICMP
}

struct target.Target {
    + ID
    + Endpoint
    + Config
    + IsActive
    + ProbeIntervalSec
}
struct probe.ProbeResult {
    + ID
    + Target
    + LatencyMs
    + ProbeTime
    + NetworkFailure
    + StatusCode
    + Meta
    + ProcessingStatus
}

interface probe.ProbeResultRepository {
    + Create(probeResult *ProbeResult)
}

interface "target.TargetRepository" {
    + Create(ctx context.Context, target *Target) (int64, error)
    + GetByID(ctx context.Context, id int64) (*Target, error)
    + GetAllActive(ctx context.Context) ([]Target, error)
    + Update(ctx context.Context, target *Target) error
    + DeleteByID(ctx context.Context, id int64) error
    + Disable(ctx context.Context, id int64) error
    + Enable(ctx context.Context, id int64) error
}

namespace repository {
    class "PostgresTargetRepository" implements "target.TargetRepository"{
        + Create(ctx context.Context, target *target.Target) (int64, error)
        + GetByID(ctx context.Context, id int64) (*target.Target, error)
        + GetAllActive(ctx context.Context) ([]target.Target, error)
        + Update(ctx context.Context, target *target.Target) error
        + DeleteByID(ctx context.Context, id int64) error
        + Disable(ctx context.Context, id int64) error
        + Enable(ctx context.Context, id int64) error
    }

    class "PostgresProbeResultRepository" implements "probe.ProbeResultRepository" {
        + Create(probeResult *probe.ProbeResult)
    }
}

"healthcheck.HealthChecker" --> "target.Target"
"healthcheck.HealthChecker" *-- "healthcheck.Scheduler"
"healthcheck.HealthChecker" *-- "healthcheck.WorkerPool"
"healthcheck.ProberRegistry" --> "target.Protocol"
"healthcheck.ProberRegistry" o-- "healthcheck.Prober"
"healthcheck.Scheduler" --> "target.Target"
"healthcheck.Scheduler" o-- "target.TargetRepository"
"healthcheck.Scheduler" o-- "healthcheck.scheduleEntry"
"healthcheck.WorkerPool" o-- "probe.ProbeResultRepository"
"healthcheck.WorkerPool" --> "target.Target"
"healthcheck.WorkerPool" o-- "healthcheck.ProberRegistry"
"healthcheck.scheduleEntry" o-- "target.Target"
"HealthChecker" *-- "HealthCheckerConfig"
target.Target --> "target.Protocol"
"target.TargetRepository" --> target.Target
"probe.ProbeResultRepository" --> probe.ProbeResult
"probe.ProbeResult" o-- target.Target
@enduml
```
