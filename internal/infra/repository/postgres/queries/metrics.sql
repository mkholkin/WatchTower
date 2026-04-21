-- name: GetStatusHistory :many
select msl.*
from monitor_status_log msl
where msl.monitor_id = $1
  and msl.end_time > $2
  and msl.start_time <= $3
order by msl.start_time desc;


-- name: GetSLAStat :one
with bounds as (
    select
        $1::uuid as monitor_id,
        $2::timestamp as period_start,
        $3::timestamp as period_end
), clipped as (
    select
        b.monitor_id,
        greatest(msl.start_time, b.period_start) as interval_start,
        least(msl.end_time, b.period_end) as interval_end,
        msl.status
    from bounds b
             left join monitor_status_log msl
                       on msl.monitor_id = b.monitor_id
                           and msl.end_time > b.period_start
                           and msl.start_time < b.period_end
), totals as (
    select
        c.monitor_id,
        coalesce(sum(
                         case
                             when c.status = 'UP' and c.interval_end > c.interval_start
                                 then extract(epoch from (c.interval_end - c.interval_start))
                             else 0
                             end
                 ), 0)::double precision as uptime_sec,
        coalesce(sum(
                         case
                             when c.status is not null
                                 and c.status <> 'UP'
                                 and c.interval_end > c.interval_start
                                 then extract(epoch from (c.interval_end - c.interval_start))
                             else 0
                             end
                 ), 0)::double precision as downtime_sec
    from clipped c
    group by c.monitor_id
)
select
    b.monitor_id,
    case
        when b.period_end <= b.period_start then 0::double precision
        else (t.uptime_sec * 100.0) / extract(epoch from (b.period_end - b.period_start))
        end as uptime_percent,
    t.downtime_sec::int as total_downtime_sec,
    b.period_start,
    b.period_end
from bounds b
         join totals t on t.monitor_id = b.monitor_id;
