# Ход выполнения TIOPO-6

[X] 1. HTTP/PostgreSQL трассировка и расширенные логи
[X] 2. Матрица испытаний и очистка
[X] 3. Расчёты и отчёты
[X] 4. Полный прогон, CI, результаты и коммит

10.11.2026: unit/race tests трассировки, регрессия Gin→pgx и 17 Python + 3 JS проверки прошли. Smoke `20261110t104949-3f0d37`: 4 режима ramp, Allure HTML сформирован; independent review пересчитал CPU и подтвердил данные. Smoke помечен skipped, в выводы о производительности не включается.

Полная серия `20261110t105253-a41b89` запущена в выделенной Colima `watchtower-lab4` (4 CPU / 6 GiB), лог `/tmp/watchtower-lab6-full.log`, артефакты `build/lab6/runs/20261110t105253-a41b89`. Команда: `DOCKER_HOST=unix:///Users/mkh0lkin/.colima/watchtower-lab4/docker.sock python3 tools/lab6/run.py --runtime-base watchtower-lab4-app:20261011t214228-cfa496 --postgres-image watchtower-lab4-postgres:20261011t214228-cfa496`. Кэшированные образы использованы из-за DNS Docker Hub в VM; текущий API пересобран без сети. Clock correction services временно остановлены; восстановить `lima-guestagent.service systemd-timesyncd.service`, затем остановить VM по завершении. Серия завершена: 20 опытов, 195927 запросов, 351550 spans измерительных окон, целостность без замечаний. Сохранены сравнительные таблицы, JSON, пять графиков, выводы и таблица требований. Allure содержит 20 passed и 10 вложений трасс.

Финальная проверка: 18 Python + 3 JS, исходные Go-пакеты, race-тесты трассировки и независимое ревью прошли. VM на финальной проверке остановлена; команды systemctl stop не меняли автозагрузку служб. Результаты находятся в tiopo/lab6/.
