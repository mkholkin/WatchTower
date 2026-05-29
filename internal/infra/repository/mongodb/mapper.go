package mongodb

import (
	"encoding/json"
	"fmt"

	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/target"
)

func unmarshalNetworkConfig(protocol target.Protocol, payload []byte) (target.NetworkConfig, error) {
	switch protocol {
	case target.ProtocolHTTP:
		var cfg target.HTTPConfig
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &cfg); err != nil {
				return nil, err
			}
		}
		return cfg, nil
	case target.ProtocolTCP:
		var cfg target.TCPConfig
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &cfg); err != nil {
				return nil, err
			}
		}
		return cfg, nil
	case target.ProtocolICMP:
		var cfg target.ICMPConfig
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &cfg); err != nil {
				return nil, err
			}
		}
		return cfg, nil
	default:
		return nil, fmt.Errorf("unknown target protocol: %s", protocol)
	}
}

func unmarshalContactConfig(contactType alert.ContactType, payload []byte) (alert.ContactConfig, error) {
	switch contactType {
	case alert.ContactTypeTelegram:
		var cfg alert.TelegramContactConfig
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &cfg); err != nil {
				return nil, err
			}
		}
		return cfg, nil
	default:
		return nil, fmt.Errorf("unknown contact type: %s", contactType)
	}
}

func unmarshalMaintenanceWindowConfig(windowType maintenance.WindowType, payload []byte) (maintenance.MaintenanceWindowConfig, error) {
	switch windowType {
	case maintenance.WindowTypeOneTime:
		var cfg maintenance.OneTimeMaintenanceWindowConfig
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &cfg); err != nil {
				return nil, err
			}
		}
		return cfg, nil
	case maintenance.WindowTypeManual:
		var cfg maintenance.ManualMaintenanceWindowConfig
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &cfg); err != nil {
				return nil, err
			}
		}
		return cfg, nil
	default:
		return nil, fmt.Errorf("unknown maintenance window type: %s", windowType)
	}
}

func unmarshalExpectations(protocol target.Protocol, payload []byte) (monitor.Expectations, error) {
	switch protocol {
	case target.ProtocolHTTP:
		var exp monitor.HTTPExpectations
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &exp); err != nil {
				return nil, err
			}
		}
		return exp, nil
	case target.ProtocolTCP:
		var exp monitor.TCPExpectations
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &exp); err != nil {
				return nil, err
			}
		}
		return exp, nil
	case target.ProtocolICMP:
		var exp monitor.ICMPExpectations
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &exp); err != nil {
				return nil, err
			}
		}
		return exp, nil
	default:
		return nil, fmt.Errorf("unknown protocol for expectations: %s", protocol)
	}
}
