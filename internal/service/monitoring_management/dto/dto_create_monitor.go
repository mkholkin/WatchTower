package dto

import (
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/target"
	"fmt"

	"github.com/google/uuid"
)

type CreateMonitorDTO struct {
	Label                string      `json:"label"`
	Endpoint             string      `json:"endpoint"`
	ProbeIntervalSec     int32       `json:"probe_interval_sec"`
	AlertContactIDs      []uuid.UUID `json:"alert_channel_ids"`
	MaintenanceWindowIDs []uuid.UUID `json:"maintainance_windows_ids"`
	NetworkConfig        MonitorNetworkConfig
	Expectations         MonitorExpectations
}

// MonitorNetworkConfig defines a typed protocol configuration used by monitor-related use-cases.
type MonitorNetworkConfig interface {
	Protocol() target.Protocol
}

// HTTPMonitorNetworkConfig is a strict HTTP config contract for monitor-related operations.
type HTTPMonitorNetworkConfig struct {
	Method          string            `json:"method"`
	Headers         map[string]string `json:"headers,omitempty"`
	Body            string            `json:"body,omitempty"`
	FollowRedirects bool              `json:"follow_redirects"`
}

func (HTTPMonitorNetworkConfig) Protocol() target.Protocol { return target.ProtocolHTTP }

// TCPMonitorNetworkConfig is a strict TCP config contract for monitor-related operations.
type TCPMonitorNetworkConfig struct {
	Port int `json:"port"`
}

func (TCPMonitorNetworkConfig) Protocol() target.Protocol { return target.ProtocolTCP }

// ICMPMonitorNetworkConfig is a strict ICMP config contract for monitor-related operations.
type ICMPMonitorNetworkConfig struct{}

func (ICMPMonitorNetworkConfig) Protocol() target.Protocol { return target.ProtocolICMP }

// MonitorExpectations defines typed protocol expectations for monitor-related use-cases.
type MonitorExpectations interface {
	Protocol() target.Protocol
}

// NetworkConfigMapper maps a strict protocol config DTO into domain NetworkConfig.
type NetworkConfigMapper interface {
	Map(cfg MonitorNetworkConfig) (target.NetworkConfig, error)
}

// ExpectationsMapper maps a strict expectations DTO into domain Expectations.
type ExpectationsMapper interface {
	Map(exp MonitorExpectations) (monitor.Expectations, error)
}

// HTTPMonitorExpectations is a strict HTTP expectations contract for monitor-related operations.
type HTTPMonitorExpectations struct {
	StatusCodes  []int `json:"status_code"`
	MaxLatencyMs int   `json:"max_latency_ms"`
}

func (HTTPMonitorExpectations) Protocol() target.Protocol { return target.ProtocolHTTP }

// TCPMonitorExpectations is a strict TCP expectations contract for monitor-related operations.
type TCPMonitorExpectations struct {
	MaxLatencyMs int `json:"max_latency_ms"`
}

func (TCPMonitorExpectations) Protocol() target.Protocol { return target.ProtocolTCP }

// ICMPMonitorExpectations is a strict ICMP expectations contract for monitor-related operations.
type ICMPMonitorExpectations struct {
	MaxLatencyMs         int `json:"max_latency_ms"`
	MaxPacketLossPercent int `json:"max_packet_loss_percent"`
}

func (ICMPMonitorExpectations) Protocol() target.Protocol { return target.ProtocolICMP }

// ToDomainExpectations maps use-case DTO to domain Expectations using protocol mapper map.
func (dto CreateMonitorDTO) ToDomainExpectations(mappers map[target.Protocol]ExpectationsMapper) (monitor.Expectations, error) {
	if mappers == nil {
		return nil, fmt.Errorf("expectations mappers are nil")
	}
	if dto.Expectations == nil {
		return nil, fmt.Errorf("expectations are nil")
	}

	protocol := dto.Expectations.Protocol()
	mapper, ok := mappers[protocol]
	if !ok {
		return nil, fmt.Errorf("no expectations mapper registered for protocol %s", protocol)
	}

	expectations, err := mapper.Map(dto.Expectations)
	if err != nil {
		return nil, err
	}

	return expectations, nil
}

func (dto CreateMonitorDTO) ToDomainNetworkConfig(mappers map[target.Protocol]NetworkConfigMapper) (target.NetworkConfig, error) {
	if mappers == nil {
		return nil, fmt.Errorf("network config mappers are nil")
	}
	if dto.NetworkConfig == nil {
		return nil, fmt.Errorf("network config is nil")
	}

	protocol := dto.NetworkConfig.Protocol()
	mapper, ok := mappers[protocol]
	if !ok {
		return nil, fmt.Errorf("no network config mapper registered for protocol %s", protocol)
	}

	networkConfig, err := mapper.Map(dto.NetworkConfig)
	if err != nil {
		return nil, err
	}

	return networkConfig, nil
}

// ValidateProtocolConsistency ensures network config and expectations describe the same protocol.
func (dto CreateMonitorDTO) ValidateProtocolConsistency() error {
	if dto.NetworkConfig == nil {
		return fmt.Errorf("network config is nil")
	}
	if dto.Expectations == nil {
		return fmt.Errorf("expectations are nil")
	}

	if dto.NetworkConfig.Protocol() != dto.Expectations.Protocol() {
		return fmt.Errorf(
			"protocol mismatch: network config protocol %s, expectations protocol %s",
			dto.NetworkConfig.Protocol(),
			dto.Expectations.Protocol(),
		)
	}

	return nil
}
