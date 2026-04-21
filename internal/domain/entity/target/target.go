package target

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/google/uuid"
)

type Target struct {
	ID               uuid.UUID
	Endpoint         string
	Config           NetworkConfig
	IsActive         bool
	ProbeIntervalSec int32
	ConfigHash       string
}

func NewTarget(endpoint string, probeIntervalSec int32, config NetworkConfig) (*Target, error) {
	// TODO: validate endpoint
	if probeIntervalSec <= 0 {
		return nil, wrapValidationf("invalid probe interval %d", probeIntervalSec)
	}

	if config == nil {
		return nil, wrapValidation("network config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Target{
		ID:               uuid.New(),
		Endpoint:         endpoint,
		ProbeIntervalSec: probeIntervalSec,
		Config:           config,
		IsActive:         true,
		ConfigHash:       ComputeHash(endpoint, config),
	}, nil
}

func (t *Target) UpdateProbeInterval(probeIntervalSec int32) error {
	if probeIntervalSec <= 0 {
		return wrapValidationf("invalid probe interval %d", probeIntervalSec)
	}

	t.ProbeIntervalSec = probeIntervalSec
	return nil
}

// ComputeHash computes a hash of the target's endpoint and network config.
// Targets with the same endpoint and config will have the same hash.
func ComputeHash(endpoint string, config NetworkConfig) string {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return ""
	}

	payload := endpoint + string(config.Protocol()) + string(configJSON)
	h := sha256.Sum256([]byte(payload))

	return hex.EncodeToString(h[:])
}
