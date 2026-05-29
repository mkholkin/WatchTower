package dto

import (
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/target"
	"fmt"

	"github.com/google/uuid"
)

type UpdateMonitorDTO struct {
	ID               uuid.UUID             `json:"id"`
	Label            *string               `json:"label,omitempty"`
	Endpoint         *string               `json:"endpoint,omitempty"`
	ProbeIntervalSec *int32                `json:"probe_interval_sec,omitempty"`
	Protocol         *target.Protocol      `json:"protocol,omitempty"`
	NetworkConfig    *MonitorNetworkConfig `json:"network_config,omitempty"`
	Expectations     *MonitorExpectations  `json:"expectations,omitempty"`
}

func (dto UpdateMonitorDTO) ToDomainNetworkConfig(mappers map[target.Protocol]NetworkConfigMapper) (target.NetworkConfig, error) {
	if mappers == nil {
		return nil, fmt.Errorf("network config mappers are nil")
	}
	if dto.NetworkConfig == nil {
		return nil, nil // No update
	}

	protocol := (*dto.NetworkConfig).Protocol()
	mapper, ok := mappers[protocol]
	if !ok {
		return nil, fmt.Errorf("no network config mapper registered for protocol %s", protocol)
	}

	networkConfig, err := mapper.Map(*dto.NetworkConfig)
	if err != nil {
		return nil, err
	}

	return networkConfig, nil
}

func (dto UpdateMonitorDTO) ToDomainExpectations(mappers map[target.Protocol]ExpectationsMapper) (monitor.Expectations, error) {
	if mappers == nil {
		return nil, fmt.Errorf("expectations mappers are nil")
	}
	if dto.Expectations == nil {
		return nil, nil // No update
	}

	protocol := (*dto.Expectations).Protocol()
	mapper, ok := mappers[protocol]
	if !ok {
		return nil, fmt.Errorf("no expectations mapper registered for protocol %s", protocol)
	}

	expectations, err := mapper.Map(*dto.Expectations)
	if err != nil {
		return nil, err
	}

	return expectations, nil
}
