package configs

import (
	"fmt"
	"strings"
)

type Config struct {
	Logging  LoggingConfig  `yaml:"logging"`
	Auth     AuthConfig     `yaml:"auth"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	MigrationsDir string 	`yaml:"migrations_dir"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

type AuthConfig struct {
	JWTSecret   string `yaml:"jwt_secret"`
	JWTTTLHours int    `yaml:"jwt_ttl_hours"`
}

type DatabaseConfig struct {
	Monitoring    ServiceDBConfig `yaml:"monitoring"`
	Maintenance   ServiceDBConfig `yaml:"maintenance"`
	Auth          ServiceDBConfig `yaml:"auth"`
	Metrics       ServiceDBConfig `yaml:"metrics"`
	Healthchecker ServiceDBConfig `yaml:"healthchecker"`
	Analyzer      ServiceDBConfig `yaml:"analyzer"`
	Contacts      ServiceDBConfig `yaml:"contacts"`
	Migrations	  ServiceDBConfig `yaml:"migrations"`
}

type ServiceDBConfig struct {
	Type string `yaml:"type"` // "postgres" (default) or "mongodb"
	DSN  string `yaml:"dsn"`
}

// DBType returns the database type, defaulting to "postgres" if empty.
func (c ServiceDBConfig) DBType() string {
	if c.Type == "" {
		return "postgres"
	}
	return c.Type
}

type RedisConfig struct {
	URL string `yaml:"url"`
}

func (d DatabaseConfig) DSNFor(service string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(service)) {
	case "monitoring":
		return d.Monitoring.DSN, nil
	case "maintenance":
		return d.Maintenance.DSN, nil
	case "auth":
		return d.Auth.DSN, nil
	case "metrics":
		return d.Metrics.DSN, nil
	case "healthchecker":
		return d.Healthchecker.DSN, nil
	case "analyzer":
		return d.Analyzer.DSN, nil
	case "contacts":
		return d.Contacts.DSN, nil
	case "migration":
		return d.Migrations.DSN, nil
	default:
		return "", fmt.Errorf("unknown database service: %q", service)
	}
}
