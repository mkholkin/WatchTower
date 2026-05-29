package configs

import (
	"fmt"
	"os"

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

