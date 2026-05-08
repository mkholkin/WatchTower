-- name: GetStatusHistory :many
select msl.*
from monitor_status_log msl
where msl.monitor_id = $1
  and msl.end_time > $2
  and msl.start_time <= $3
order by msl.start_time desc;

-- name: GetSLAStat :one
WITH bounds AS (
    SELECT
        $1::uuid AS monitor_id,
        -- Clamp start to the greater of requested start or the earliest possible data
        -- You can replace '1970-01-01' with a subquery if you have a specific min date
        $2::timestamptz as period_start,
--         greatest($2::timestamptz, '1970-01-01 00:00:00+00'::timestamptz) AS period_start,
        least($3::timestamptz, now())::timestamptz AS period_end
),
     clipped AS (
         SELECT
             b.monitor_id,
             -- Use the clamped bounds to define the window
             greatest(msl.start_time, b.period_start) AS interval_start,
             least(msl.end_time, b.period_end) AS interval_end,
             msl.status
         FROM bounds b
                  LEFT JOIN monitor_status_log msl
                            ON msl.monitor_id = b.monitor_id
                                AND msl.end_time > b.period_start
                                AND msl.start_time < b.period_end
     ),
     totals AS (
         SELECT
             b.monitor_id,
             COALESCE(SUM(
                              CASE
                                  WHEN c.status = 'UP' AND c.interval_end > c.interval_start
                                      THEN EXTRACT(EPOCH FROM (c.interval_end - c.interval_start))
                                  ELSE 0
                                  END
                      ), 0)::double precision AS uptime_sec,
             COALESCE(SUM(
                              CASE
                                  WHEN c.status = 'DOWN' AND c.interval_end > c.interval_start
                                      THEN EXTRACT(EPOCH FROM (c.interval_end - c.interval_start))
                                  ELSE 0
                                  END
                      ), 0)::double precision AS downtime_sec
         FROM bounds b
                  LEFT JOIN clipped c ON b.monitor_id = c.monitor_id
         GROUP BY b.monitor_id
     )
SELECT
    b.monitor_id,
    CASE
        WHEN (t.uptime_sec + t.downtime_sec) <= 0 THEN 0::double precision
        ELSE (t.uptime_sec * 100.0) / (t.uptime_sec + t.downtime_sec)
        END AS uptime_percent,
    t.downtime_sec::bigint AS total_downtime_sec,
    b.period_start,
    b.period_end
FROM bounds b
         JOIN totals t ON t.monitor_id = b.monitor_id;
