# Lab 6 measured overhead

Run: `20261110t105253-a41b89`; smoke: `False`.

CPU-seconds are raw cumulative CPU nanosecond differences at measurement boundaries, linearly interpolated between adjacent samples. RAM is working set; warmup and cooldown are excluded. Collector overhead is reported separately. Δ compares the same profile and repetition with baseline.

| Profile | Variant / repetition | Component | CPU s | Δ CPU s | Δ CPU % | RAM min / median / max MiB | Δ median MiB | RPS median | Unexpected % | Intentional % |
|---|---|---|---:|---:|---:|---|---:|---:|---:|---:|
| ramp | baseline / 1 | api | 80.488 | 0.000 | 0.000 | 11.004 / 15.375 / 20.289 | 0.000 | 71.00 | 7.38 | 0.00 |
| ramp | baseline / 1 | collector | 0.402 | 0.000 | 0.000 | 8.285 / 8.297 / 8.547 | 0.000 | 71.00 | 7.38 | 0.00 |
| ramp | trace / 1 | api | 81.138 | 0.650 | 0.808 | 11.160 / 15.498 / 20.348 | 0.123 | 59.40 | 8.89 | 0.00 |
| ramp | trace / 1 | collector | 0.432 | 0.030 | 7.586 | 8.285 / 8.297 / 8.547 | 0.000 | 59.40 | 8.89 | 0.00 |
| ramp | debug / 1 | api | 82.484 | 1.996 | 2.480 | 10.848 / 15.121 / 19.117 | -0.254 | 60.10 | 8.17 | 0.00 |
| ramp | debug / 1 | collector | 0.447 | 0.045 | 11.279 | 8.273 / 8.289 / 8.535 | -0.008 | 60.10 | 8.17 | 0.00 |
| ramp | trace-debug / 1 | api | 81.000 | 0.512 | 0.636 | 10.922 / 15.762 / 20.188 | 0.387 | 55.90 | 8.78 | 0.00 |
| ramp | trace-debug / 1 | collector | 0.395 | -0.007 | -1.721 | 8.281 / 8.297 / 8.547 | 0.000 | 55.90 | 8.78 | 0.00 |
| stress | trace / 1 | api | 73.967 | -2.001 | -2.635 | 11.105 / 15.348 / 20.801 | -1.695 | 36.00 | 26.39 | 0.00 |
| stress | trace / 1 | collector | 0.473 | 0.158 | 50.029 | 8.281 / 8.293 / 8.543 | 0.000 | 36.00 | 26.39 | 0.00 |
| stress | debug / 1 | api | 76.545 | 0.576 | 0.759 | 10.879 / 16.012 / 20.863 | -1.031 | 36.00 | 24.75 | 0.00 |
| stress | debug / 1 | collector | 0.403 | 0.087 | 27.720 | 8.297 / 8.309 / 8.555 | 0.016 | 36.00 | 24.75 | 0.00 |
| stress | trace-debug / 1 | api | 75.999 | 0.030 | 0.040 | 11.031 / 15.773 / 20.652 | -1.270 | 36.00 | 27.41 | 0.00 |
| stress | trace-debug / 1 | collector | 0.434 | 0.119 | 37.574 | 8.281 / 8.293 / 8.543 | 0.000 | 36.00 | 27.41 | 0.00 |
| stress | baseline / 1 | api | 75.968 | 0.000 | 0.000 | 11.090 / 17.043 / 21.086 | 0.000 | 36.00 | 24.12 | 0.00 |
| stress | baseline / 1 | collector | 0.315 | 0.000 | 0.000 | 8.281 / 8.293 / 8.543 | 0.000 | 36.00 | 24.12 | 0.00 |
| soak-0 | debug / 1 | api | 75.281 | -0.031 | -0.041 | 11.367 / 13.199 / 17.477 | -0.867 | 36.00 | 0.00 | 0.00 |
| soak-0 | debug / 1 | collector | 0.846 | -0.053 | -5.856 | 8.281 / 8.297 / 8.551 | 0.008 | 36.00 | 0.00 | 0.00 |
| soak-0 | trace-debug / 1 | api | 75.186 | -0.125 | -0.166 | 11.656 / 13.703 / 17.340 | -0.363 | 36.00 | 0.00 | 0.00 |
| soak-0 | trace-debug / 1 | collector | 0.902 | 0.003 | 0.379 | 8.285 / 8.297 / 8.551 | 0.008 | 36.00 | 0.00 | 0.00 |
| soak-0 | baseline / 1 | api | 75.312 | 0.000 | 0.000 | 11.516 / 14.066 / 17.320 | 0.000 | 36.00 | 0.00 | 0.00 |
| soak-0 | baseline / 1 | collector | 0.898 | 0.000 | 0.000 | 8.277 / 8.289 / 8.543 | 0.000 | 36.00 | 0.00 | 0.00 |
| soak-0 | trace / 1 | api | 75.594 | 0.283 | 0.375 | 11.809 / 13.773 / 17.195 | -0.293 | 36.00 | 0.00 | 0.00 |
| soak-0 | trace / 1 | collector | 0.874 | -0.024 | -2.669 | 8.273 / 8.289 / 8.543 | 0.000 | 36.00 | 0.00 | 0.00 |
| soak-10 | trace-debug / 1 | api | 75.862 | 0.921 | 1.229 | 11.836 / 13.980 / 17.578 | 0.676 | 36.00 | 0.00 | 10.00 |
| soak-10 | trace-debug / 1 | collector | 0.642 | -0.337 | -34.408 | 8.277 / 8.293 / 8.547 | -0.008 | 36.00 | 0.00 | 10.00 |
| soak-10 | baseline / 1 | api | 74.941 | 0.000 | 0.000 | 11.672 / 13.305 / 17.414 | 0.000 | 36.00 | 0.00 | 10.00 |
| soak-10 | baseline / 1 | collector | 0.978 | 0.000 | 0.000 | 8.285 / 8.301 / 8.555 | 0.000 | 36.00 | 0.00 | 10.00 |
| soak-10 | trace / 1 | api | 75.396 | 0.455 | 0.607 | 11.496 / 13.855 / 17.375 | 0.551 | 36.00 | 0.00 | 10.00 |
| soak-10 | trace / 1 | collector | 0.908 | -0.070 | -7.164 | 8.277 / 8.297 / 8.547 | -0.004 | 36.00 | 0.00 | 10.00 |
| soak-10 | debug / 1 | api | 75.462 | 0.521 | 0.695 | 11.629 / 13.504 / 17.074 | 0.199 | 36.00 | 0.00 | 10.00 |
| soak-10 | debug / 1 | collector | 0.866 | -0.112 | -11.415 | 8.281 / 8.301 / 8.551 | 0.000 | 36.00 | 0.00 | 10.00 |
| soak-20 | baseline / 1 | api | 74.967 | 0.000 | 0.000 | 11.207 / 13.855 / 17.062 | 0.000 | 36.00 | 0.00 | 20.00 |
| soak-20 | baseline / 1 | collector | 0.901 | 0.000 | 0.000 | 8.281 / 8.297 / 8.551 | 0.000 | 36.00 | 0.00 | 20.00 |
| soak-20 | trace / 1 | api | 75.172 | 0.205 | 0.273 | 11.504 / 14.078 / 17.078 | 0.223 | 36.00 | 0.00 | 20.00 |
| soak-20 | trace / 1 | collector | 0.844 | -0.057 | -6.329 | 8.281 / 8.297 / 8.551 | 0.000 | 36.00 | 0.00 | 20.00 |
| soak-20 | debug / 1 | api | 79.522 | 4.555 | 6.076 | 11.746 / 13.539 / 17.387 | -0.316 | 36.00 | 0.00 | 20.00 |
| soak-20 | debug / 1 | collector | 1.073 | 0.172 | 19.042 | 8.273 / 8.293 / 8.547 | -0.004 | 36.00 | 0.00 | 20.00 |
| soak-20 | trace-debug / 1 | api | 75.425 | 0.458 | 0.611 | 11.539 / 13.379 / 17.008 | -0.477 | 36.00 | 0.00 | 20.00 |
| soak-20 | trace-debug / 1 | collector | 0.982 | 0.080 | 8.889 | 8.277 / 8.293 / 8.543 | -0.004 | 36.00 | 0.00 | 20.00 |

The workload is closed-loop: CPU changes must be read together with RPS and errors. One run does not estimate run-to-run uncertainty or establish causality. Traces and HTTP diagnostics include file/logging costs; OS and Docker daemon remain shared. Smoke runs are diagnostics, not final measurements.

## Проверка реализации

| Проверка | Результат |
|---|---|
| Полная матрица 5 × 4 | 20 опытов, код 0, нарушений целостности нет |
| Allure | 20 passed; 10 вложений Chrome Trace; HTML сформирован |
| Исходные Go-пакеты проекта | 30 пакетов с тестами — passed |
| `go test -race ./internal/observability ./cmd/api` | passed |
| `make lab6-check` | 18 Python + 3 JS — passed |
| Независимый пересчёт CPU API и collector | Совпадает с отчётом |
| Связь SQL → HTTP | 257721 дочерний span, нарушений идентификаторов и временных границ нет |
| Конфигурации и измеряемый код | Совпадают с baseline своего профиля; хэши исходников проверены |
| Очистка контейнеров, сетей, томов и конфигов | Успешна для всех 20 опытов |
| Ресурсы VM | На финальной проверке Colima VM остановлена |

Go-пакеты проверены командой `go test ./cmd/... ./configs/... ./internal/... ./pkg/... ./benchmarks/... ./tests/... ./tools/...` с доступом к локальному TCP для SMTP-проверки. Полный локальный прогон: `20261110t105253-a41b89`. GitHub workflow подготовлен, но удалённый запуск в этой работе не выполнялся.

Выводы: [analysis.md](analysis.md). Методика: [README.md](README.md). Таблица требований: [requirements.md](requirements.md).
