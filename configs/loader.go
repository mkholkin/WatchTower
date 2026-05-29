package configs

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads YAML config from path and unmarshals it into Config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config yaml: %w", err)
	}

	return &cfg, nil
}

// LoadFromEnv reads config values from environment variables.
// Expected variables:
// WT_LOGGING_LEVEL, WT_LOGGING_FORMAT, WT_LOGGING_OUTPUT,
// WT_AUTH_JWT_SECRET, WT_AUTH_JWT_TTL_HOURS,
// WT_DATABASE_MONITORING_DSN, WT_DATABASE_MONITORING_TYPE,
// WT_DATABASE_MAINTENANCE_DSN, WT_DATABASE_MAINTENANCE_TYPE,
// WT_DATABASE_AUTH_DSN, WT_DATABASE_AUTH_TYPE,
// WT_DATABASE_METRICS_DSN, WT_DATABASE_METRICS_TYPE,
// WT_DATABASE_HEALTHCHECKER_DSN, WT_DATABASE_HEALTHCHECKER_TYPE,
// WT_DATABASE_ANALYZER_DSN, WT_DATABASE_ANALYZER_TYPE,
// WT_DATABASE_CONTACTS_DSN, WT_DATABASE_CONTACTS_TYPE,
// WT_REDIS_URL.
func LoadFromEnv() (*Config, error) {
	var cfg Config

	setStringFromEnv(&cfg.Logging.Level, "WT_LOGGING_LEVEL")
	setStringFromEnv(&cfg.Logging.Format, "WT_LOGGING_FORMAT")
	setStringFromEnv(&cfg.Logging.Output, "WT_LOGGING_OUTPUT")

	setStringFromEnv(&cfg.Auth.JWTSecret, "WT_AUTH_JWT_SECRET")
	if raw, ok := os.LookupEnv("WT_AUTH_JWT_TTL_HOURS"); ok && strings.TrimSpace(raw) != "" {
		jwtTTLHours, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("parse WT_AUTH_JWT_TTL_HOURS: %w", err)
		}
		cfg.Auth.JWTTTLHours = jwtTTLHours
	}

	setStringFromEnv(&cfg.Database.Monitoring.DSN, "WT_DATABASE_MONITORING_DSN")
	setStringFromEnv(&cfg.Database.Monitoring.Type, "WT_DATABASE_MONITORING_TYPE")
	setStringFromEnv(&cfg.Database.Maintenance.DSN, "WT_DATABASE_MAINTENANCE_DSN")
	setStringFromEnv(&cfg.Database.Maintenance.Type, "WT_DATABASE_MAINTENANCE_TYPE")
	setStringFromEnv(&cfg.Database.Auth.DSN, "WT_DATABASE_AUTH_DSN")
	setStringFromEnv(&cfg.Database.Auth.Type, "WT_DATABASE_AUTH_TYPE")
	setStringFromEnv(&cfg.Database.Metrics.DSN, "WT_DATABASE_METRICS_DSN")
	setStringFromEnv(&cfg.Database.Metrics.Type, "WT_DATABASE_METRICS_TYPE")
	setStringFromEnv(&cfg.Database.Healthchecker.DSN, "WT_DATABASE_HEALTHCHECKER_DSN")
	setStringFromEnv(&cfg.Database.Healthchecker.Type, "WT_DATABASE_HEALTHCHECKER_TYPE")
	setStringFromEnv(&cfg.Database.Analyzer.DSN, "WT_DATABASE_ANALYZER_DSN")
	setStringFromEnv(&cfg.Database.Analyzer.Type, "WT_DATABASE_ANALYZER_TYPE")
	setStringFromEnv(&cfg.Database.Contacts.DSN, "WT_DATABASE_CONTACTS_DSN")
	setStringFromEnv(&cfg.Database.Contacts.Type, "WT_DATABASE_CONTACTS_TYPE")

	setStringFromEnv(&cfg.Redis.URL, "WT_REDIS_URL")

	return &cfg, nil
}

func setStringFromEnv(dst *string, key string) {
	if value, ok := os.LookupEnv(key); ok {
		*dst = value
	}
}

