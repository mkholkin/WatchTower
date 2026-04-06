-- name: CreateMonitor :exec
INSERT INTO "monitor" (id, target_id, user_login, label, is_active, probe_interval_sec, expectations, current_status,
                       created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetMonitorByID :one
SELECT m.*, t.*, u.password_hash,
       COALESCE(
           (SELECT jsonb_agg(ac.*)
            FROM "alert_contact" ac
            JOIN "monitor_alert_contact" mac ON ac.id = mac.contact_id
            WHERE mac.monitor_id = m.id), '[]'::jsonb
       ) AS alert_contacts,
       COALESCE(
           (SELECT jsonb_agg(mw.*)
            FROM "maintenance_window" mw
            JOIN "maintenance_window_monitor" mwm ON mw.id = mwm.window_id
            WHERE mwm.monitor_id = m.id), '[]'::jsonb
       ) AS maintenance_windows
FROM "monitor" m
         JOIN "target" t ON m.target_id = t.id
         JOIN "user" u ON m.user_login = u.login
WHERE m.id = $1;

-- name: UpdateMonitor :exec
UPDATE "monitor"
SET target_id          = $2,
    user_login         = $3,
    label              = $4,
    is_active          = $5,
    probe_interval_sec = $6,
    expectations       = $7,
    current_status     = $8,
    last_evaluated_at  = $9
WHERE id = $1;

-- name: DeleteMonitorByID :exec
DELETE
FROM "monitor"
WHERE id = $1;

-- name: GetAllMonitorsByUser :many
SELECT m.*, t.*, u.password_hash,
       COALESCE(
           (SELECT jsonb_agg(ac.*)
            FROM "alert_contact" ac
            JOIN "monitor_alert_contact" mac ON ac.id = mac.contact_id
            WHERE mac.monitor_id = m.id), '[]'::jsonb
       ) AS alert_contacts,
       COALESCE(
           (SELECT jsonb_agg(mw.*)
            FROM "maintenance_window" mw
            JOIN "maintenance_window_monitor" mwm ON mw.id = mwm.window_id
            WHERE mwm.monitor_id = m.id), '[]'::jsonb
       ) AS maintenance_windows
FROM "monitor" m
         JOIN "target" t ON m.target_id = t.id
         JOIN "user" u ON m.user_login = u.login
WHERE m.user_login = $1;

-- name: GetAllMonitorsByTargetID :many
SELECT m.*, t.*, u.password_hash,
       COALESCE(
           (SELECT jsonb_agg(ac.*)
            FROM "alert_contact" ac
            JOIN "monitor_alert_contact" mac ON ac.id = mac.contact_id
            WHERE mac.monitor_id = m.id), '[]'::jsonb
       ) AS alert_contacts,
       COALESCE(
           (SELECT jsonb_agg(mw.*)
            FROM "maintenance_window" mw
            JOIN "maintenance_window_monitor" mwm ON mw.id = mwm.window_id
            WHERE mwm.monitor_id = m.id), '[]'::jsonb
       ) AS maintenance_windows
FROM "monitor" m
         JOIN "target" t ON m.target_id = t.id
         JOIN "user" u ON m.user_login = u.login
WHERE m.target_id = $1;

-- name: GetMonitorsToEvaluate :many
SELECT m.*, t.*, u.password_hash,
       COALESCE(
           (SELECT jsonb_agg(ac.*)
            FROM "alert_contact" ac
            JOIN "monitor_alert_contact" mac ON ac.id = mac.contact_id
            WHERE mac.monitor_id = m.id), '[]'::jsonb
       ) AS alert_contacts,
       COALESCE(
           (SELECT jsonb_agg(mw.*)
            FROM "maintenance_window" mw
            JOIN "maintenance_window_monitor" mwm ON mw.id = mwm.window_id
            WHERE mwm.monitor_id = m.id), '[]'::jsonb
       ) AS maintenance_windows
FROM "monitor" m
         JOIN "target" t ON m.target_id = t.id
         JOIN "user" u ON m.user_login = u.login
WHERE m.is_active = TRUE
  AND m.last_evaluated_at + (m.probe_interval_sec || ' seconds') :: interval <= NOW()
  AND m.target_id = ANY (@target_ids::uuid[]);

-- name: BulkUpdateEvaluation :exec
UPDATE "monitor" AS m
SET current_status    = c.current_status,
    last_evaluated_at = c.last_evaluated_at
FROM (
    SELECT unnest(@ids::uuid[]) AS id,
           unnest(@statuses::status_type[]) AS current_status,
           unnest(@evaluated_ats::TIMESTAMP[]) AS last_evaluated_at
) AS c
WHERE m.id = c.id
  AND m.is_active = TRUE;

-- name: AddAlertContactToMonitor :exec
INSERT INTO "monitor_alert_contact" (monitor_id, contact_id)
VALUES ($1, $2);

-- name: RemoveAlertContactFromMonitor :exec
DELETE
FROM "monitor_alert_contact"
WHERE monitor_id = $1
  AND contact_id = $2;

-- name: EnableMonitor :exec
UPDATE "monitor"
SET is_active = TRUE
WHERE id = $1;

-- name: DisableMonitor :exec
UPDATE "monitor"
SET is_active = FALSE
WHERE id = $1;

