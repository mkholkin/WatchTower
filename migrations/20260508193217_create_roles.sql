-- +goose Up
SELECT 'up SQL query';

create user monitoring_svc with password 'password';
grant SELECT, INSERT, UPDATE, DELETE on table monitor to monitoring_svc;
grant SELECT, INSERT, UPDATE on table target to monitoring_svc;
grant SELECT on table "user" to monitoring_svc;
grant select on table alert_contact to monitoring_svc;
grant select, INSERT, DELETE on table monitor_alert_contact to monitoring_svc;
grant select on table maintenance_window to monitoring_svc;
grant select on table maintenance_window_monitor to monitoring_svc;

create user auth_svc with password 'password';
grant SELECT, INSERT, UPDATE on table "user" to auth_svc;

create user healthchecker with password 'password';
grant SELECT on table monitor to healthchecker;
grant SELECT, INSERT on TABLE probe_result to healthchecker;
grant select on table target to healthchecker;

create user analyzer with password 'password';
grant select on table "user" to analyzer;
grant select, update on table monitor to analyzer;
grant select on table alert_contact to analyzer;
grant select on table monitor_alert_contact to analyzer;
grant select on table maintenance_window to analyzer;
grant select on table maintenance_window_monitor to analyzer;
grant SELECT, UPDATE on table probe_result to analyzer;
grant select on table target to analyzer;

create user alert_contacts_svc with password 'password';
grant select on table "user" to alert_contacts_svc;
grant SELECT, UPDATE, INSERT, DELETE on table alert_contact to alert_contacts_svc;

create user metrics_query_svc with password 'password';
grant SELECT on table probe_result to metrics_query_svc;
grant select on table alert_contact to metrics_query_svc;
grant select on table monitor_alert_contact to metrics_query_svc;
grant select on table maintenance_window to metrics_query_svc;
grant select on table maintenance_window_monitor to metrics_query_svc;
grant select on table "user" to metrics_query_svc;
grant select on table monitor to metrics_query_svc;
grant select on table target to metrics_query_svc;
grant select on table monitor_status_log to metrics_query_svc;

create user maintenance_svc with password 'password';
grant select on table "user" to maintenance_svc;
grant select on table monitor to maintenance_svc;
grant select on table alert_contact to maintenance_svc;
grant select on table monitor_alert_contact to maintenance_svc;
grant select on table target to maintenance_svc;
grant SELECT, UPDATE, INSERT, DELETE on table maintenance_window to maintenance_svc;
grant select, INSERT, DELETE on table maintenance_window_monitor to maintenance_svc;


-- +goose Down
SELECT 'down SQL query';

drop owned by monitoring_svc;
drop user if exists monitoring_svc;

drop owned by auth_svc;
drop user if exists auth_svc;

drop owned by healthchecker;
drop user if exists healthchecker;

drop owned by analyzer;
drop user if exists analyzer;

drop owned by alert_contacts_svc;
drop user if exists alert_contacts_svc;

drop owned by metrics_query_svc;
drop user if exists metrics_query_svc;

drop owned by maintenance_svc;
drop user if exists maintenance_svc;
