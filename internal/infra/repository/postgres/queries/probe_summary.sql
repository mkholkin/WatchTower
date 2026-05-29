-- name: GetProbeSummaryByMonitorID :many
SELECT m.id                         AS monitor_id,
       pr.latency_ms                AS latency_ms,
       pr.probe_time                AS probe_time,
       msl.status                   AS monitor_status,
       pr.status_code               AS status_code,
       pr.network_failure           AS network_failure,
       pr.error_message             AS failure_reason
FROM "probe_result" pr
         JOIN monitor m
              on m.target_id = pr.target_id
         JOIN monitor_status_log msl
              on m.id = msl.monitor_id
                  and msl.start_time <= pr.probe_time
                  and pr.probe_time < msl.end_time
WHERE m.id = $1
  and pr.processing_status = 'PROCESSED'
ORDER BY pr.probe_time DESC
LIMIT $2;

-- name: GetProbeSummaryByMonitorIDForPeriod :many
SELECT m.id                           AS monitor_id,
       pr.latency_ms                  AS latency_ms,
       pr.probe_time                  AS probe_time,
       msl.status                     AS monitor_status,
       COALESCE(pr.status_code, 0)    AS status_code,
       pr.network_failure             AS network_failure,
       COALESCE(pr.error_message, '') AS failure_reason
FROM "probe_result" pr
         JOIN monitor m
              ON m.target_id = pr.target_id
         JOIN monitor_status_log msl
              ON m.id = msl.monitor_id
                  AND msl.start_time <= pr.probe_time
                  AND pr.probe_time < msl.end_time
WHERE m.id = $1
  AND pr.processing_status = 'PROCESSED'
  AND pr.probe_time >= $2
  AND pr.probe_time <= $3
ORDER BY pr.probe_time DESC;

