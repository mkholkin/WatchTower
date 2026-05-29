-- 	// GetByIDBulk retrieves multiple alert contacts by their IDs. Returns a slice of found contacts.
-- 	GetByIDBulk(ctx context.Context, ids []uuid.UUID) ([]alert.Contact, error)

-- name: CreateAlertContact :exec
INSERT INTO "alert_contact" (id, user_login, type, label, config, is_active)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetAlertContactByID :one
SELECT sqlc.embed(a), sqlc.embed(u)
FROM "alert_contact" a
         JOIN "user" u ON a.user_login = u.login
WHERE id = $1;

-- name: UpdateAlertContact :exec
UPDATE "alert_contact"
SET user_login = $2,
    type       = $3,
    label      = $4,
    config     = $5
WHERE id = $1;

-- name: DeleteAlertContactByID :exec
DELETE
FROM "alert_contact"
WHERE id = $1;

-- name: GetAlertContactsByUserLogin :many
SELECT sqlc.embed(a), sqlc.embed(u)
FROM "alert_contact" a
         JOIN "user" u ON a.user_login = u.login
WHERE a.user_login = $1;

-- name: EnableAlertContact :exec
UPDATE "alert_contact"
SET is_active = TRUE
WHERE id = $1
  AND is_active = FALSE;

-- name: DisableAlertContact :exec
UPDATE "alert_contact"
SET is_active = FALSE
WHERE id = $1
  AND is_active = TRUE;

-- name: GetAlertContactsByIDBulk :many
SELECT sqlc.embed(a), sqlc.embed(u)
FROM "alert_contact" a
         JOIN "user" u ON a.user_login = u.login
WHERE a.id = ANY (@ids::uuid[]);
