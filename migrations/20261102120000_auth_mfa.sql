-- +goose Up
CREATE TABLE auth_mfa (
    login varchar PRIMARY KEY REFERENCES "user" (login) ON DELETE CASCADE,
    email text NOT NULL,
    failed_attempts integer NOT NULL DEFAULT 0 CHECK (failed_attempts >= 0),
    locked boolean NOT NULL DEFAULT false,
    password_changed_at timestamptz NOT NULL DEFAULT now(),
    token_version bigint NOT NULL DEFAULT 1,
    challenge_id text UNIQUE,
    otp_hash bytea,
    otp_expires_at timestamptz,
    recovery_hash bytea,
    recovery_expires_at timestamptz
);

-- +goose Down
DROP TABLE auth_mfa;
