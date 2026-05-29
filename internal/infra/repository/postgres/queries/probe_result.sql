-- name: CreateProbeResult :exec
INSERT INTO "probe_result" (id, target_id, probe_time, latency_ms, status_code, network_failure, error_message, meta)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetUnprocessedProbeResults :many
SELECT *
FROM "probe_result"
WHERE processing_status = 'new'
ORDER BY probe_time ASC
LIMIT $1;

-- name: BulkUpdateProbeResultStatus :exec
UPDATE "probe_result"
SET processing_status = $1
WHERE id = ANY (@ids::uuid[]);
