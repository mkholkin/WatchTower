-- +goose Up
SELECT 'up SQL query';

CREATE TYPE "protocol_type" AS ENUM (
    'HTTP',
    'TCP',
    'ICMP'
    );

CREATE TYPE "status_type" AS ENUM (
    'UP',
    'DOWN',
    'MAINTENANCE',
    'UNKNOWN'
    );

CREATE TYPE "maintenance_type" AS ENUM (
    'ONCE',
    'MANUAL'
    );

CREATE TYPE "contact_type" AS ENUM (
    'TELEGRAM'
    );

CREATE TYPE "processing_status_type" AS ENUM (
    'NEW',
    'PROCESSED',
    'CANCELLED'
    );

CREATE TABLE "user"
(
    "login"         varchar PRIMARY KEY,
    "password_hash" varchar NOT NULL
);

CREATE TABLE "target"
(
    "id"                 uuid PRIMARY KEY,
    "signature_hash"     varchar       NOT NULL UNIQUE,
    "protocol"           protocol_type NOT NULL,
    "is_active"          bool          NOT NULL DEFAULT TRUE,
    "endpoint"           varchar       NOT NULL,
    "network_config"     jsonb         NOT NULL,
    "probe_interval_sec" int           NOT NULL CHECK ( probe_interval_sec > 0)
);

CREATE TABLE "probe_result"
(
    "id"                uuid PRIMARY KEY,
    "target_id"         uuid                   NOT NULL REFERENCES target (id) ON DELETE CASCADE,
    "probe_time"        timestamptz            NOT NULL,
    "latency_ms"        int                    NOT NULL CHECK (latency_ms >= 0),
    "status_code"       int,
    "network_failure"   boolean                NOT NULL,
    "error_message"     text,
    "meta"              jsonb,
    "processing_status" processing_status_type NOT NULL DEFAULT 'NEW'

        CONSTRAINT no_status_on_network_failure CHECK (
            network_failure = TRUE AND status_code IS NULL
                OR
            network_failure = FALSE AND status_code IS NOT NULL
            )
);

CREATE TABLE "monitor"
(
    "id"                 uuid PRIMARY KEY,
    "target_id"          uuid        NOT NULL REFERENCES target (id),
    "user_login"         varchar     NOT NULL REFERENCES "user" (login) ON DELETE CASCADE,
    "label"              varchar     NOT NULL CHECK ( label <> '' ),
    "is_active"          boolean     NOT NULL DEFAULT TRUE,
    "probe_interval_sec" int         NOT NULL CHECK ( probe_interval_sec > 0),
    "expectations"       jsonb       NOT NULL,
    "current_status"     status_type NOT NULL DEFAULT 'UNKNOWN',
    "last_evaluated_at"  timestamptz,
    "created_at"         timestamptz   NOT NULL DEFAULT NOW()
);

CREATE TABLE "monitor_status_log"
(
    "id"         bigserial PRIMARY KEY,
    "monitor_id" uuid        NOT NULL REFERENCES monitor (id) ON DELETE CASCADE,
    "status"     status_type NOT NULL,
    "start_time" timestamptz   NOT NULL,
    "end_time"   timestamptz   NOT NULL DEFAULT 'infinity'
);

CREATE TABLE "maintenance_window"
(
    "id"          uuid PRIMARY KEY,
    "user_login"  varchar          NOT NULL REFERENCES "user" (login) ON DELETE CASCADE,
    "title"       varchar          NOT NULL CHECK ( title <> '' ),
    "description" text CHECK ( description <> '' ),
    "type"        maintenance_type NOT NULL,
    "config"      jsonb            NOT NULL
);

CREATE TABLE "maintenance_window_monitor"
(
    "monitor_id" uuid NOT NULL REFERENCES monitor (id) ON DELETE CASCADE,
    "window_id"  uuid NOT NULL REFERENCES maintenance_window (id) ON DELETE CASCADE,
    PRIMARY KEY ("window_id", "monitor_id")
);

CREATE TABLE "alert_contact"
(
    "id"         uuid PRIMARY KEY,
    "user_login" varchar      NOT NULL REFERENCES "user" (login) ON DELETE CASCADE,
    "type"       contact_type NOT NULL,
    "label"      varchar      NOT NULL CHECK ( label <> '' ),
    "config"     jsonb        NOT NULL,
    "is_active"  boolean      NOT NULL DEFAULT TRUE
);

CREATE TABLE "monitor_alert_contact"
(
    "monitor_id" uuid NOT NULL REFERENCES monitor (id) ON DELETE CASCADE,
    "contact_id" uuid NOT NULL REFERENCES alert_contact (id) ON DELETE CASCADE,
    PRIMARY KEY ("monitor_id", "contact_id")
);


-- +goose Down
SELECT 'down SQL query';

DROP TABLE IF EXISTS "monitor_alert_contact";
DROP TABLE IF EXISTS "alert_contact";
DROP TABLE IF EXISTS "maintenance_window_monitor";
DROP TABLE IF EXISTS "maintenance_window";
DROP TABLE IF EXISTS "monitor_status_log";
DROP TABLE IF EXISTS "monitor";
DROP TABLE IF EXISTS "probe_result";
DROP TABLE IF EXISTS "target";
DROP TABLE IF EXISTS "user";

DROP TYPE IF EXISTS "protocol_type";
DROP TYPE IF EXISTS "status_type";
DROP TYPE IF EXISTS "maintenance_type";
DROP TYPE IF EXISTS "contact_type";
DROP TYPE IF EXISTS "processing_status_type";
