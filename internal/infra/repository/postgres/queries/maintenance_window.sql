-- name: CreateMaintenanceWindow :exec
INSERT INTO "maintenance_window" (id, user_login, title, description, type, config)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetMaintenanceWindowByID :one
SELECT sqlc.embed(m), sqlc.embed(u)
FROM "maintenance_window" m
         JOIN "user" u ON m.user_login = u.login
WHERE id = $1;

-- name: UpdateMaintenanceWindow :exec
UPDATE "maintenance_window"
SET user_login  = $2,
    title       = $3,
    description = $4,
    type        = $5,
    config      = $6
WHERE id = $1;

-- name: DeleteMaintenanceWindowByID :exec
DELETE
FROM "maintenance_window"
WHERE id = $1;

-- name: GeMaintenanceWindowsByIDBulk :many
SELECT sqlc.embed(m), sqlc.embed(u)
FROM "maintenance_window" m
         JOIN "user" u ON m.user_login = u.login
WHERE m.id = ANY (@ids::uuid[]);

-- name: GetMaintenanceWindowsByUserLogin :many
SELECT sqlc.embed(m), sqlc.embed(u)
FROM "maintenance_window" m
         JOIN "user" u ON m.user_login = u.login
WHERE m.user_login = $1
ORDER BY m.title;

-- name: LinkMonitor :exec
INSERT INTO "maintenance_window_monitor" (monitor_id, window_id)
VALUES ($1, $2);

-- name: UnlinkMonitor :exec
DELETE
FROM "maintenance_window_monitor"
WHERE monitor_id = $1
  AND window_id = $2;
