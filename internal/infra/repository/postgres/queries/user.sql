-- name: CreateUser :exec
INSERT INTO "user" (login, password_hash)
VALUES ($1, $2);

-- name: GetUserByLogin :one
SELECT login, password_hash
FROM "user"
WHERE login = $1;