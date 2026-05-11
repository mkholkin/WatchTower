-- +goose Up
SELECT 'up SQL query';

CREATE SCHEMA partman;
CREATE EXTENSION pg_partman SCHEMA partman;

SELECT partman.create_parent(
               p_parent_table := 'public.probe_result',
               p_control := 'probe_time',
               p_type := 'range',
               p_interval := 'hourly',
               p_premake := 3
       );



-- +goose Down
SELECT 'down SQL query';
