-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'auth_svc') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON auth_mfa TO auth_svc;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'auth_svc') THEN
        REVOKE SELECT, INSERT, UPDATE, DELETE ON auth_mfa FROM auth_svc;
    END IF;
END $$;
-- +goose StatementEnd
