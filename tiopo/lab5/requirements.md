| Требование | Тест, файл или команда |
|---|---|
| BDD E2E аутентификации | `tests/lab5/features/authentication.feature`; `make lab5` |
| Второй фактор по email | `@REQ-MFA`, Mailpit и `internal/infra/notification/smtp_auth.go` |
| Ограничение числа попыток | `@REQ-LOCK`, `@REQ-CONCURRENCY` |
| Восстановление после блокировки | `@REQ-RECOVERY`; `/auth/mfa/recover`, `/auth/mfa/reset` |
| Плановая смена пароля | `@REQ-ROTATION`, `@REQ-REVOKE`; `/auth/mfa/password` |
| Одноразовость и срок секретов | `@REQ-OTP`, `@REQ-RECOVERY`, `internal/service/mfa/postgres_test.go` |
| Создание и утилизация технического пользователя | `tests/lab5/features/environment.py`, `support.py` |
| Отсутствие сохранённых паролей для CI | `tools/lab5/run.py`, `tests/lab5/features/support.py`: runtime secrets |
| Интеграция в CI/CD | `.github/workflows/tiopo-lab5.yml` |
| Allure и cleanup при ошибке | `tools/lab5/run.py`, `tools/lab5/test_run.py`; `make lab5-check` |
