# WatchTower WebLab № 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (- [ ]) syntax for tracking.

**Goal:** Подготовить в ветке WEB-1 полный комплект базовой WebLab № 1, переиспользовав материалы WatchTower и не изменяя код приложения.

**Architecture:** README.md служит единым отчётом и связывает требования, сценарии, диаграммы, стек, эскизы и ADR. Существующие PlantUML, DBML и PNG сохраняются как редактируемые исходники; добавляются только недостающая третья BPMN-диаграмма и короткий журнал работы.

**Tech Stack:** Markdown, PlantUML, DBML, Git и существующие Go/React/Docker-артефакты проекта.

**Spec:** docs/superpowers/specs/2026-09-20-watchtower-weblab1-design.md

## Global Constraints

- Выполнять только основную часть WebLab № 1; требования ИИ-трека исключены.
- Работать в ветке WEB-1, созданной от main.
- Не изменять бизнес-логику, API, миграции, конфигурацию развёртывания и UI.
- Не трогать build/, tools/, tiopo/, allure-results/, .DS_Store, LAB1_RUN=shuffle-2026 и PROJECT_CONTEXT.md.
- Описывать как реализованные только HTTP-проверки и Telegram-уведомления; TCP, ICMP, SMTP и WebSocket не выдавать за работающие функции.
- Основной отчёт хранить в существующем README.md.

## Review Focus

- Устаревшие утверждения о SMTP, WebSocket, TCP или ICMP: диаграммы должны показывать только фактические runtime-интеграции.
- Неполное покрытие 11 пунктов: каждый пункт должен иметь раздел или ссылку в README.md.
- Менее трёх нелинейных BPMN-процессов: каталог должен содержать три содержательные диаграммы с ветвлениями.
- Непроверяемые нефункциональные требования: производительность, надёжность и безопасность должны содержать численные критерии.
- Случайное изменение кода или пользовательских артефактов: итоговый diff ограничен README.md и docs/.

---

### Task 1: Привести диаграммы к фактической системе

**Files:**
- Modify: docs/puml/c4/l2.puml
- Modify: docs/puml/c4/l3.puml
- Modify: docs/puml/bpmn/bpmn-new-monitor.puml
- Modify: docs/puml/bpmn/bpmn-target-check.puml
- Create: docs/puml/bpmn/bpmn-maintenance-window.puml

**Interfaces:**
- Consumes: факты из internal/app/app.go, HTTP-prober, Telegram-provider и спецификации.
- Produces: три непротиворечивые BPMN и C4 L2/L3, на которые ссылается Task 2.

- [ ] **Step 1: Зафиксировать исходные расхождения**

Run:

~~~bash
rg -n 'SMTP|WSS|HTTP/TCP/ICMP|Telegram, SMTP' docs/puml/c4 docs/puml/bpmn
~~~

Expected: совпадения в l2.puml и l3.puml, которые требуется исправить.

- [ ] **Step 2: Исправить C4 L2 и L3**

В l2.puml указать JSON/HTTP(S) между SPA и backend, HTTP для проверки ресурсов, Telegram API для уведомлений и Redis как кэш последних ProbeSummary.

В l3.puml заменить HTTP/TCP/ICMP Targets, Telegram, SMTP и связанные протоколы на фактические HTTP и Telegram. Сохранить существующую структуру модульного backend-приложения.

- [ ] **Step 3: Уточнить существующие BPMN**

В bpmn-new-monitor.puml исправить «спецефичные» на «специфичные» и добавить отрицательную ветку валидации, возвращающую ошибку без создания монитора.

В bpmn-target-check.puml исправить синтаксис действия с ответом, показать приоритет активного окна обслуживания, уведомлять при любом изменении статуса и исправить «для мониторе».

- [ ] **Step 4: Добавить третий нелинейный BPMN-процесс**

Создать bpmn-maintenance-window.puml с дорожками «Пользователь», «Сервис обслуживания» и «Анализатор». Процесс содержит выбор разового или ручного окна, проверку конфигурации, ошибочную ветку, привязку мониторов и ветвление оценки в MAINTENANCE либо обычный UP/DOWN.

- [ ] **Step 5: Проверить структуру PlantUML и устаревшие интеграции**

Run:

~~~bash
for f in docs/puml/c4/l2.puml docs/puml/c4/l3.puml docs/puml/bpmn/*.puml; do test "$(rg -c '^@start' "$f")" -eq 1 && test "$(rg -c '^@end' "$f")" -eq 1 || exit 1; done
test "$(find docs/puml/bpmn -name '*.puml' | wc -l | tr -d ' ')" -ge 3
! rg -n 'SMTP|WSS|HTTP/TCP/ICMP|Telegram, SMTP' docs/puml/c4/l2.puml docs/puml/c4/l3.puml
~~~

Expected: exit code 0; у каждого файла одна открывающая и одна закрывающая директива, BPMN-файлов не меньше трёх, запрещённых описаний нет.

- [ ] **Step 6: Проверить diff и закоммитить**

Run:

~~~bash
git diff --check -- docs/puml
git diff -- docs/puml
git add docs/puml/c4/l2.puml docs/puml/c4/l3.puml docs/puml/bpmn/bpmn-new-monitor.puml docs/puml/bpmn/bpmn-target-check.puml docs/puml/bpmn/bpmn-maintenance-window.puml
git commit -m "docs: align WebLab 1 diagrams with implementation"
~~~

Expected: проверка формата проходит; коммит содержит только перечисленные PlantUML-файлы.

### Task 2: Оформить единый отчёт в README

**Files:**
- Modify: README.md

**Interfaces:**
- Consumes: диаграммы Task 1, docs/db.dbml, img/img1.png, img/img2.png, код и проектный контекст.
- Produces: единый отчёт по всем 11 пунктам WebLab № 1.

- [ ] **Step 1: Сохранить полезную продуктовую часть**

Оставить название, краткое описание, изображения Dashboard и Monitor Details и рабочую команду Docker Compose. Удалить обещания Email и полного подавления уведомлений при обслуживании.

- [ ] **Step 2: Добавить цель и функциональные требования**

Описать проблему ручного контроля доступности и реализованный минимум: регистрация и вход; CRUD и включение мониторов; периодические HTTP-проверки; статусы UP, DOWN, MAINTENANCE, UNKNOWN; история, SLA, Telegram-контакты; разовые и ручные окна обслуживания.

- [ ] **Step 3: Добавить измеримые нефункциональные требования**

Зафиксировать как критерии приёмки, а не измеренные результаты:

- производительность: при 50 запросах/с в течение 10 минут p95 чтения списка и карточки монитора не больше 500 мс, HTTP 5xx не больше 1%;
- надёжность: после перезапуска backend возобновляет планирование сохранённых активных целей не позднее 10 секунд без потери конфигурации;
- безопасность: 100% защищённых операций отклоняют отсутствующий или неверный JWT, пароли хранятся как bcrypt-хеши, секреты не попадают в ответы и логи.

- [ ] **Step 4: Добавить три пользовательских сценария**

Для «создать монитор», «разобрать инцидент» и «запланировать обслуживание» описать актора, предусловия, основной поток, альтернативную или ошибочную ветку и проверяемый результат. Учесть дедупликацию Target, историю/SLA, уведомления, разовые и ручные окна.

- [ ] **Step 5: Связать отчёт с артефактами**

Добавить ссылки на Use Case, три BPMN, ER, DBML, C4 L1/L2/L3 и встроить img/img1.png и img/img2.png. Пояснить, что PlantUML и DBML — редактируемые исходники.

- [ ] **Step 6: Описать стек и два ADR**

Указать Go 1.26, Gin, oapi-codegen, pgx/sqlc, Goose, Watermill GoChannel, PostgreSQL, Redis, React 19, TypeScript, Axios, Tailwind, Recharts и Docker Compose.

Добавить ADR-001 о модульном монолите и ADR-002 о PostgreSQL с Redis. Для каждого указать контекст, решение, альтернативы и последствия.

- [ ] **Step 7: Проверить полноту README**

Run:

~~~bash
rg -n '^## (Цель|Функциональные требования|Нефункциональные требования|Пользовательские сценарии|Диаграммы|Технологический стек|Архитектурные решения|Запуск)' README.md
rg -n 'производительность|надёжность|безопасность|ADR-001|ADR-002|BPMN|C4|ER-диаграмм|DBML' README.md
rg -n '50 запрос|500 мс|10 секунд|100%' README.md
~~~

Expected: первая команда находит восемь разделов, вторая — каждый контрольный термин, третья — численные критерии приёмки.

- [ ] **Step 8: Проверить diff и закоммитить**

Run:

~~~bash
git diff --check -- README.md
git diff -- README.md
git add README.md
git commit -m "docs: complete WebLab 1 report"
~~~

Expected: коммит изменяет только README.md.

### Task 3: Добавить журнал и выполнить итоговую приёмку

**Files:**
- Create: docs/labs/WEB-1.md
- Verify: README.md
- Verify: docs/puml/**/*.puml
- Verify: docs/db.dbml
- Verify: img/img1.png
- Verify: img/img2.png

**Interfaces:**
- Consumes: готовые диаграммы и отчёт Tasks 1–2.
- Produces: краткая запись о работе и воспроизводимые доказательства проверки.

- [ ] **Step 1: Создать краткий журнал**

Записать ветку WEB-1, назначение работы, переиспользованные материалы, исправленные расхождения, новые материалы, команды проверок и фактические результаты. Не включать историю взаимодействия с ИИ.

- [ ] **Step 2: Проверить локальные ссылки**

Run:

~~~bash
for p in docs/puml/use-case.puml docs/puml/er-diagram.puml docs/db.dbml docs/puml/c4/l1.puml docs/puml/c4/l2.puml docs/puml/c4/l3.puml docs/puml/bpmn/bpmn-new-monitor.puml docs/puml/bpmn/bpmn-target-check.puml docs/puml/bpmn/bpmn-maintenance-window.puml img/img1.png img/img2.png; do test -f "$p" || exit 1; done
~~~

Expected: exit code 0.

- [ ] **Step 3: Проверить границы и формат diff**

Run:

~~~bash
git diff --check main --
test -z "$(git diff --name-only main -- | rg -v '^(README\.md|docs/)')"
~~~

Expected: изменены только README.md и файлы внутри docs/.

- [ ] **Step 4: Проверить сборку API**

Run:

~~~bash
go build -o /tmp/watchtower-api-check ./cmd/api
~~~

Expected: exit code 0; бинарник создан вне репозитория.

- [ ] **Step 5: Сверить 11 пунктов задания**

В docs/labs/WEB-1.md отметить фактическое наличие цели; функциональных и трёх измеримых нефункциональных требований; Use Case; трёх BPMN; трёх сценариев; ER; стека; DBML; C4 L1/L2/L3; двух эскизов; двух ADR.

- [ ] **Step 6: Закоммитить журнал**

Run:

~~~bash
git add docs/labs/WEB-1.md
git commit -m "docs: record WebLab 1 verification"
git status --short --branch
~~~

Expected: коммит содержит только docs/labs/WEB-1.md; остаются лишь исходные пользовательские untracked-артефакты из Global Constraints.
