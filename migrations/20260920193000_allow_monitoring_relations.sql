-- +goose Up
GRANT INSERT ON TABLE maintenance_window_monitor TO monitoring_svc;

-- +goose Down
REVOKE INSERT ON TABLE maintenance_window_monitor FROM monitoring_svc;
