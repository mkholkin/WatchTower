# Lab 4 measured results

Run: `20261011t214228-cfa496`; smoke: `False`.

All latencies include client blocking, connection setup and response transfer. Setup/warmup/cooldown excluded. Percentiles use nearest rank over individual requests. Throughput uses fixed 5-second windows, including zero-traffic windows.

| Profile | Requests | Intentional errors % | Unexpected % | p50 ms | p75 | p90 | p95 | p99 | RPS median | RPS p95 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| ramp | 9341 | 0.00 | 10.87 | 3.0 | 117.0 | 2000.0 | 2001.0 | 2001.0 | 48.2 | 128.0 |
| stress | 3843 | 0.00 | 53.34 | 2000.0 | 2001.0 | 2001.0 | 2001.0 | 2003.0 | 32.0 | 50.6 |
| soak-0 | 10650 | 0.00 | 0.00 | 1.0 | 1.0 | 56.0 | 69.0 | 97.0 | 36.0 | 38.0 |
| soak-10 | 10670 | 10.00 | 0.00 | 1.0 | 1.0 | 57.0 | 66.0 | 94.0 | 36.0 | 38.0 |
| soak-20 | 10660 | 20.00 | 0.00 | 1.0 | 1.0 | 56.0 | 69.0 | 101.0 | 36.0 | 38.0 |

## Load phases

| Profile / phase | VUs | Seconds | Requests | p95 ms | Unexpected % | RPS median | SLO |
|---|---:|---:|---:|---:|---:|---:|---|
| ramp / vus-1 | 1 | 20 | 220 | 69 | 0.00 | 11.0 | True |
| ramp / vus-2 | 2 | 20 | 370 | 68 | 0.00 | 18.0 | True |
| ramp / vus-4 | 4 | 20 | 740 | 68 | 0.00 | 37.0 | True |
| ramp / vus-8 | 8 | 20 | 1400 | 104 | 0.00 | 71.0 | True |
| ramp / vus-16 | 16 | 20 | 1787 | 507 | 0.00 | 95.6 | False |
| ramp / vus-32 | 32 | 20 | 2553 | 1144 | 0.00 | 127.8 | False |
| ramp / vus-64 | 64 | 20 | 1442 | 2001 | 15.46 | 76.6 | False |
| ramp / vus-128 | 128 | 20 | 829 | 2001 | 95.54 | 42.2 | False |
| stress / baseline | 4 | 30 | 1070 | 68 | 0.00 | 36.0 | True |
| stress / overload | 128 | 45 | 1928 | 2001 | 98.29 | 47.0 | False |
| stress / recovery | 4 | 90 | 845 | 2001 | 18.34 | 1.6 | False |
| soak-0 / steady | 4 | 300 | 10650 | 69 | 0.00 | 36.0 | True |
| soak-10 / steady | 4 | 300 | 10670 | 66 | 0.00 | 36.0 | True |
| soak-20 / steady | 4 | 300 | 10660 | 69 | 0.00 | 36.0 | True |

Recovery evidence: `{"first_healthy_s": 70, "confirmed_s": 85, "baseline_meets_slo": true, "overload_observed": true, "baseline_good_rps": 36.0}`.


## Resources

CPU: percent of one core. RAM: working set, excluding inactive file cache. Disk: cgroup block I/O deltas (cache hits do not count as physical reads). GPU is not used.

| Profile / component | CPU min / median / max % | RAM min / median / max MiB | Read min / median / max KiB/s | Write min / median / max KiB/s |
|---|---|---|---|---|
| ramp / api | 5.95 / 41.47 / 100.76 | 11.18 / 15.34 / 25.47 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| ramp / postgres | 0.47 / 2.93 / 7.52 | 55.66 / 76.08 / 81.11 | 0.00 / 0.00 / 0.00 | 0.00 / 55.90 / 832.84 |
| ramp / redis | 0.17 / 0.97 / 2.08 | 3.47 / 3.50 / 3.77 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| ramp / probe | 0.01 / 0.01 / 0.47 | 9.19 / 9.19 / 9.22 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| ramp / load | 0.45 / 1.15 / 5.04 | 40.45 / 49.04 / 61.25 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| ramp / collector | 0.16 / 0.20 / 0.83 | 8.29 / 8.30 / 8.55 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| stress / api | 14.15 / 100.00 / 103.20 | 11.41 / 40.20 / 55.12 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| stress / postgres | 0.30 / 2.22 / 5.72 | 52.38 / 68.30 / 76.43 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 507.62 |
| stress / redis | 0.18 / 0.86 / 2.04 | 3.47 / 3.49 / 4.50 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| stress / probe | 0.00 / 0.01 / 0.15 | 9.18 / 9.18 / 9.21 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| stress / load | 0.15 / 0.89 / 8.35 | 39.07 / 59.54 / 60.30 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| stress / collector | 0.14 / 0.17 / 0.69 | 8.28 / 8.29 / 8.54 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| soak-0 / api | 12.74 / 24.81 / 31.01 | 11.54 / 13.77 / 17.14 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| soak-0 / postgres | 0.67 / 2.95 / 8.37 | 55.14 / 70.29 / 75.75 | 0.00 / 0.00 / 0.00 | 31.99 / 96.00 / 521.10 |
| soak-0 / redis | 0.28 / 1.06 / 2.19 | 3.47 / 3.49 / 4.80 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| soak-0 / probe | 0.00 / 0.01 / 0.37 | 9.18 / 9.18 / 9.18 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| soak-0 / load | 0.53 / 0.94 / 1.62 | 12.25 / 15.05 / 16.12 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| soak-0 / collector | 0.14 / 0.19 / 0.69 | 8.28 / 8.30 / 8.55 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| soak-10 / api | 10.04 / 24.50 / 31.15 | 11.50 / 13.59 / 17.26 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| soak-10 / postgres | 0.74 / 2.89 / 7.56 | 52.54 / 69.11 / 77.26 | 0.00 / 0.00 / 28.05 | 0.00 / 95.97 / 523.61 |
| soak-10 / redis | 0.26 / 1.09 / 2.31 | 3.47 / 3.49 / 4.55 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| soak-10 / probe | 0.01 / 0.01 / 0.33 | 9.18 / 9.18 / 9.18 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| soak-10 / load | 0.49 / 0.92 / 2.05 | 12.39 / 15.05 / 16.21 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| soak-10 / collector | 0.14 / 0.20 / 0.68 | 8.27 / 8.28 / 8.53 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| soak-20 / api | 9.00 / 24.94 / 31.05 | 10.99 / 13.75 / 17.42 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| soak-20 / postgres | 0.74 / 2.82 / 6.89 | 51.98 / 69.74 / 78.38 | 0.00 / 0.00 / 0.00 | 15.95 / 63.90 / 482.80 |
| soak-20 / redis | 0.39 / 1.10 / 2.23 | 3.46 / 3.49 / 4.80 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| soak-20 / probe | 0.00 / 0.01 / 0.26 | 9.18 / 9.18 / 9.18 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |
| soak-20 / load | 0.56 / 0.97 / 2.11 | 12.34 / 15.02 / 16.35 | 0.00 / 0.00 / 83.35 | 0.00 / 0.00 / 0.00 |
| soak-20 / collector | 0.14 / 0.20 / 0.61 | 8.27 / 8.29 / 8.54 | 0.00 / 0.00 / 0.00 | 0.00 / 0.00 / 0.00 |

Raw data, per-operation and per-phase summaries, and graphs are under `profiles/<name>/`. SLO: p95 ≤ 500 ms and unexpected responses ≤ 1%. Recovery requires three consecutive 5-second SLO-compliant windows at ≥ 80% of baseline successful RPS. A null recovery time means recovery was not confirmed within the observation interval.

These observations describe a closed user workload on this machine. Five minutes can expose short-term resource trends but cannot prove absence of long-term leaks. A single run per load profile does not estimate between-run variability.
