-- +goose Up
UPDATE auth_mfa SET locked = false WHERE locked = true;

-- +goose Down
-- Previously locked accounts cannot be reconstructed safely.
SELECT 1;
