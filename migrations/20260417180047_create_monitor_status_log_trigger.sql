-- +goose Up
SELECT 'up SQL query';

-- +goose StatementBegin
create or replace function update_monitor_status_log() returns trigger as
$$
begin
    if old.current_status != new.current_status then
        update monitor_status_log
        set end_time = new.last_evaluated_at
        where monitor_id = new.id
          and end_time = 'infinity'::timestamp;
        insert into monitor_status_log(monitor_id, status, start_time, end_time)
        values (new.id, new.current_status, new.last_evaluated_at, 'infinity'::timestamp);
    end if;
    return new;
end;
$$ language plpgsql;
-- +goose StatementEnd

create trigger monitor_status_log_trigger
    after update of current_status, last_evaluated_at
    on monitor
    for each row
execute function update_monitor_status_log();

-- +goose Down
SELECT 'down SQL query';

drop trigger if exists monitor_status_log_trigger on monitor;
drop function if exists update_monitor_status_log();
