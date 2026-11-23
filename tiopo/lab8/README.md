# ЛР №8: внешняя RDAP-интеграция

WatchTower получает сведения о регистрации домена HTTP-монитора: домен, дату регистрации и срок окончания регистрации. Внешний сервис — Verisign RDAP для `.com`, публичный HTTPS API без ключа доступа. Для согласования с преподавателем выбран этот сервис.

Контракт внешнего сервиса описан в [RFC 9082](https://www.rfc-editor.org/rfc/rfc9082.html), [RFC 9083](https://www.rfc-editor.org/rfc/rfc9083.html) и [документации Verisign](https://www.verisign.com/news-insights/registration-data-access-protocol/help/). Применяемая часть контракта и фикстура находятся в `api/contracts/rdap/`.

## Сценарий пользователя

Пользователь регистрируется, входит, создаёт HTTP-монитор домена `.com` и вызывает `GET /monitors/{monitorId}/registration` с Bearer-токеном. Приложение проверяет владельца монитора, извлекает его домен и обращается к RDAP. Для `www.example.com` запрашиваются сведения об `example.com`. Контракт нового endpoint — [api/rdap.openapi.yaml](../../api/rdap.openapi.yaml).

Ответ содержит `domain`, `registered_at`, `expires_at`, `source_url`. Отсутствующая дата возвращается как `null`. Неподходящий домен даёт 400; отсутствующая запись — 404; ошибка провайдера — 502; ограничение частоты — 503; таймаут — 504. Отсутствующий токен, чужой и отсутствующий монитор отклоняются до внешнего запроса.

## Переключение провайдера

В конфигурации приложения:

```yaml
rdap:
  enabled: true
  base_url: http://rdap-mock:8089/com/v1
  timeout_ms: 750
```

Для реального сервиса заменить `base_url` на `https://rdap.verisign.com/com/v1`, `timeout_ms` — на `10000`. Бизнес-код и HTTP-сценарий одинаковы. По умолчанию интеграция выключена; при включении без адреса и таймаута используются Verisign и 5000 мс. Клиент делает один запрос, ограничивает ответ 1 МиБ и использует стандартную проверку TLS.

## Проверка и полный прогон

Нужны Docker Engine с Compose, 4 CPU, 6 GiB RAM, Python 3.10+, Go 1.26+ и Node.js. Первый запуск собирает runtime и PostgreSQL; интернет нужен для сборки и реального E2E.

```sh
make lab8-check
make lab8-smoke
make lab8
```

Отдельные режимы:

```sh
make lab8-mock
make lab8-real
make lab8-load
```

`make lab8` выполняет E2E с mock, полный нагрузочный профиль с mock и E2E с реальным Verisign. Приватные конфиги и JWT создаются на время контура; автоматический прогон удаляет контейнеры, сеть и тома. Артефакты находятся в `build/lab8/runs/<run-id>/`, путь последнего запуска — `build/lab8/latest.txt`. Общий Allure: `allure-report/index.html`; результаты этапов — `summary.json`, `results.md`, `manifest.json`.

Для локальных кэшированных образов ЛР №4:

```sh
DOCKER_CONTEXT=colima-watchtower-lab4 python3 tools/lab8/run.py \
  --runtime-base watchtower-lab4-app:20261011t214228-cfa496 \
  --postgres-image watchtower-lab4-postgres:20261011t214228-cfa496
```

Runner пересобирает текущий API. Контекст Docker собирается по списку каталогов исходного кода; его хэши сохраняются в `source-sha256.json`.

## Демонстрация

```sh
make lab8-demo-mock
python3 tools/lab8/demo_request.py
make lab8-down
make lab8-demo-real
python3 tools/lab8/demo_request.py
make lab8-down
```

Swagger доступен на `http://localhost:3088/swagger`, спецификация RDAP — на `http://localhost:3088/openapi-rdap.yaml`. В mock-режиме счётчики видны на `http://localhost:8089/__admin/stats`. Demo оставляет один монитор и хранит его временный токен в `demo-session.json` с правами `0600`; `demo_request.py` выводит сведения о домене. Перед сменой режима выполняется `make lab8-down`.

## Нагрузка и E2E

Нагрузка использует профиль ЛР №4 `soak-0`: 15 секунд прогрева и 300 секунд при 4 VU. В цикл из десяти исходных HTTP-операций добавлен один запрос регистрации домена; число пользователей и паузы сохранены. Фоновый HTTP-пробник обращается к локальному контейнеру с DNS-алиасом `example.com`. RDAP-запросы поступают в mock. Отчёт сохраняет задержки, RPS, ошибки, ресурсы шести компонентов и отдельные счётчики mock. Число upstream-запросов должно покрывать все измеренные RDAP-операции.

Mock воспроизводит успешный ответ, 404, 429, 503, повреждённый JSON, задержку и отсутствие срока регистрации. E2E проверяет точные даты фикстуры, права доступа и отсутствие лишних повторных запросов. Реальный E2E делает один RDAP-запрос и проверяет домен, формат дат и адрес источника. Нагрузка направляется на mock согласно [условиям Verisign](https://www.verisign.com/legal-center/rdap-terms/).

В CI сначала выполняется статический gate ЛР №7, затем проверки клиента/mock/runner и полный сценарий ЛР №8. Allure и исходные данные публикуются через `if: always()`.

[Результаты](results.md). [Таблица требований](requirements.md).
