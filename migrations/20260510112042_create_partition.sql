-- +goose Up
SELECT 'up SQL query';

CREATE SCHEMA IF NOT EXISTS partman;
CREATE EXTENSION IF NOT EXISTS pg_partman SCHEMA partman;
CREATE EXTENSION IF NOT EXISTS pg_cron;

SELECT partman.create_parent(
               p_parent_table := 'public.probe_result',
               p_control := 'probe_time',
               p_type := 'range',
               p_interval := '1 hour',
               p_premake := 3
       );


SELECT cron.schedule('partman_maintenance', '@hourly', $$CALL partman.run_maintenance()$$);

-- +goose Down
SELECT 'down SQL query';

DROP EXTENSION IF EXISTS pg_partman;
DROP EXTENSION IF EXISTS pg_cron;
DROP SCHEMA IF EXISTS partman;
