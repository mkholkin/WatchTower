-- name: CreateTarget :exec
INSERT INTO "target" (id, signature_hash, protocol, endpoint, network_config, is_active, probe_interval_sec)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetTargetByID :one
SELECT *
FROM "target"
WHERE id = $1;

-- name: UpdateTarget :exec
UPDATE "target"
SET signature_hash     = $2,
    protocol           = $3,
    endpoint           = $4,
    network_config     = $5,
    is_active          = $6,
    probe_interval_sec = $7
WHERE id = $1;

-- name: DeleteTargetByID :exec
DELETE
FROM "target"
WHERE id = $1;

-- name: UpdateTargetProbeInterval :exec
UPDATE "target"
SET probe_interval_sec = $2
WHERE id = $1;

-- name: GetTargetByHash :one
SELECT *
FROM "target"
WHERE signature_hash = $1;

-- name: GetAllActiveTargets :many
SELECT *
FROM "target"
WHERE is_active = TRUE;

-- name: DisableTarget :exec
UPDATE "target"
SET is_active = FALSE
WHERE id = $1;

-- name: EnableTarget :exec
UPDATE "target"
SET is_active = TRUE
WHERE id = $1;
