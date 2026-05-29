package postgres

import (
	"encoding/json"
	"fmt"

	alert "WatchTower/internal/domain/entity/alert_contact"
	"WatchTower/internal/domain/entity/maintenance"
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/infra/repository/postgres/sqlcgen"
	"WatchTower/pkg/mapper"
)

type monitorTypeMapper struct {
	toDBStatusRegistry                mapper.Mapper[monitor.Status, sqlcgen.StatusType, monitor.Status]
	toDomainStatusRegistry            mapper.Mapper[sqlcgen.StatusType, monitor.Status, sqlcgen.StatusType]
	toDBExpectationsRegistry          mapper.Mapper[monitor.Expectations, []byte, target.Protocol]
	toDomainExpectationsRegistry      mapper.Mapper[[]byte, monitor.Expectations, target.Protocol]
	toDomainTargetProtocolRegistry    mapper.Mapper[sqlcgen.ProtocolType, target.Protocol, sqlcgen.ProtocolType]
	toDomainTargetConfigRegistry      mapper.Mapper[[]byte, target.NetworkConfig, target.Protocol]
	toDomainContactTypeRegistry       mapper.Mapper[sqlcgen.ContactType, alert.ContactType, sqlcgen.ContactType]
	toDomainContactConfigRegistry     mapper.Mapper[[]byte, alert.ContactConfig, alert.ContactType]
	toDomainWindowTypeRegistry        mapper.Mapper[sqlcgen.MaintenanceType, maintenance.WindowType, sqlcgen.MaintenanceType]
	toDomainWindowConfigTypeRegistry  mapper.Mapper[[]byte, maintenance.MaintenanceWindowConfig, maintenance.WindowType]
}

func newMonitorTypeMapper() *monitorTypeMapper {
	toDBStatusMap := mapper.New[monitor.Status, sqlcgen.StatusType, monitor.Status]()
	toDBStatusMap.Register(monitor.StatusUp, func(_ monitor.Status) (sqlcgen.StatusType, error) {
		return sqlcgen.StatusTypeUP, nil
	})
	toDBStatusMap.Register(monitor.StatusDown, func(_ monitor.Status) (sqlcgen.StatusType, error) {
		return sqlcgen.StatusTypeDOWN, nil
	})
	toDBStatusMap.Register(monitor.StatusMaintenance, func(_ monitor.Status) (sqlcgen.StatusType, error) {
		return sqlcgen.StatusTypeMAINTENANCE, nil
	})
	toDBStatusMap.Register(monitor.StatusUnknown, func(_ monitor.Status) (sqlcgen.StatusType, error) {
		return sqlcgen.StatusTypeUNKNOWN, nil
	})

	toDomainStatusMap := mapper.New[sqlcgen.StatusType, monitor.Status, sqlcgen.StatusType]()
	toDomainStatusMap.Register(sqlcgen.StatusTypeUP, func(_ sqlcgen.StatusType) (monitor.Status, error) {
		return monitor.StatusUp, nil
	})
	toDomainStatusMap.Register(sqlcgen.StatusTypeDOWN, func(_ sqlcgen.StatusType) (monitor.Status, error) {
		return monitor.StatusDown, nil
	})
	toDomainStatusMap.Register(sqlcgen.StatusTypeMAINTENANCE, func(_ sqlcgen.StatusType) (monitor.Status, error) {
		return monitor.StatusMaintenance, nil
	})
	toDomainStatusMap.Register(sqlcgen.StatusTypeUNKNOWN, func(_ sqlcgen.StatusType) (monitor.Status, error) {
		return monitor.StatusUnknown, nil
	})

	toDBExpectationsMap := mapper.New[monitor.Expectations, []byte, target.Protocol]()
	toDBExpectationsMap.Register(target.ProtocolHTTP, func(v monitor.Expectations) ([]byte, error) {
		httpExp, ok := v.(monitor.HTTPExpectations)
		if !ok {
			if httpExpPtr, ok := v.(*monitor.HTTPExpectations); ok && httpExpPtr != nil {
				httpExp = *httpExpPtr
			} else {
				return nil, fmt.Errorf("unexpected expectations type for HTTP: %T", v)
			}
		}
		return json.Marshal(httpExp)
	})
	toDBExpectationsMap.Register(target.ProtocolTCP, func(v monitor.Expectations) ([]byte, error) {
		tcpExp, ok := v.(monitor.TCPExpectations)
		if !ok {
			if tcpExpPtr, ok := v.(*monitor.TCPExpectations); ok && tcpExpPtr != nil {
				tcpExp = *tcpExpPtr
			} else {
				return nil, fmt.Errorf("unexpected expectations type for TCP: %T", v)
			}
		}
		return json.Marshal(tcpExp)
	})
	toDBExpectationsMap.Register(target.ProtocolICMP, func(v monitor.Expectations) ([]byte, error) {
		icmpExp, ok := v.(monitor.ICMPExpectations)
		if !ok {
			if icmpExpPtr, ok := v.(*monitor.ICMPExpectations); ok && icmpExpPtr != nil {
				icmpExp = *icmpExpPtr
			} else {
				return nil, fmt.Errorf("unexpected expectations type for ICMP: %T", v)
			}
		}
		return json.Marshal(icmpExp)
	})

	toDomainExpectationsMap := mapper.New[[]byte, monitor.Expectations, target.Protocol]()
	toDomainExpectationsMap.Register(target.ProtocolHTTP, func(payload []byte) (monitor.Expectations, error) {
		var exp monitor.HTTPExpectations
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &exp); err != nil {
				return nil, err
			}
		}
		return exp, nil
	})
	toDomainExpectationsMap.Register(target.ProtocolTCP, func(payload []byte) (monitor.Expectations, error) {
		var exp monitor.TCPExpectations
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &exp); err != nil {
				return nil, err
			}
		}
		return exp, nil
	})
	toDomainExpectationsMap.Register(target.ProtocolICMP, func(payload []byte) (monitor.Expectations, error) {
		var exp monitor.ICMPExpectations
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &exp); err != nil {
				return nil, err
			}
		}
		return exp, nil
	})

	toDomainTargetProtocolMap := mapper.New[sqlcgen.ProtocolType, target.Protocol, sqlcgen.ProtocolType]()
	toDomainTargetProtocolMap.Register(sqlcgen.ProtocolTypeHTTP, func(_ sqlcgen.ProtocolType) (target.Protocol, error) {
		return target.ProtocolHTTP, nil
	})
	toDomainTargetProtocolMap.Register(sqlcgen.ProtocolTypeTCP, func(_ sqlcgen.ProtocolType) (target.Protocol, error) {
		return target.ProtocolTCP, nil
	})
	toDomainTargetProtocolMap.Register(sqlcgen.ProtocolTypeICMP, func(_ sqlcgen.ProtocolType) (target.Protocol, error) {
		return target.ProtocolICMP, nil
	})

	toDomainTargetConfigMap := mapper.New[[]byte, target.NetworkConfig, target.Protocol]()
	toDomainTargetConfigMap.Register(target.ProtocolHTTP, func(payload []byte) (target.NetworkConfig, error) {
		var cfg target.HTTPConfig
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &cfg); err != nil {
				return nil, err
			}
		}
		return cfg, nil
	})
	toDomainTargetConfigMap.Register(target.ProtocolTCP, func(payload []byte) (target.NetworkConfig, error) {
		var cfg target.TCPConfig
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &cfg); err != nil {
				return nil, err
			}
		}
		return cfg, nil
	})
	toDomainTargetConfigMap.Register(target.ProtocolICMP, func(payload []byte) (target.NetworkConfig, error) {
		var cfg target.ICMPConfig
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &cfg); err != nil {
				return nil, err
			}
		}
		return cfg, nil
	})

	toDomainContactTypeMap := mapper.New[sqlcgen.ContactType, alert.ContactType, sqlcgen.ContactType]()
	toDomainContactTypeMap.Register(sqlcgen.ContactTypeTELEGRAM, func(_ sqlcgen.ContactType) (alert.ContactType, error) {
		return alert.ContactTypeTelegram, nil
	})

	toDomainContactConfigMap := mapper.New[[]byte, alert.ContactConfig, alert.ContactType]()
	toDomainContactConfigMap.Register(alert.ContactTypeTelegram, func(payload []byte) (alert.ContactConfig, error) {
		var cfg alert.TelegramContactConfig
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &cfg); err != nil {
				return nil, err
			}
		}
		return cfg, nil
	})

	toDomainWindowTypeMap := mapper.New[sqlcgen.MaintenanceType, maintenance.WindowType, sqlcgen.MaintenanceType]()
	toDomainWindowTypeMap.Register(sqlcgen.MaintenanceTypeONCE, func(_ sqlcgen.MaintenanceType) (maintenance.WindowType, error) {
		return maintenance.WindowTypeOneTime, nil
	})
	toDomainWindowTypeMap.Register(sqlcgen.MaintenanceTypeMANUAL, func(_ sqlcgen.MaintenanceType) (maintenance.WindowType, error) {
		return maintenance.WindowTypeManual, nil
	})

	toDomainWindowConfigMap := mapper.New[[]byte, maintenance.MaintenanceWindowConfig, maintenance.WindowType]()
	toDomainWindowConfigMap.Register(maintenance.WindowTypeOneTime, func(payload []byte) (maintenance.MaintenanceWindowConfig, error) {
		var cfg maintenance.OneTimeMaintenanceWindowConfig
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &cfg); err != nil {
				return nil, err
			}
		}
		return cfg, nil
	})
	toDomainWindowConfigMap.Register(maintenance.WindowTypeManual, func(payload []byte) (maintenance.MaintenanceWindowConfig, error) {
		var cfg maintenance.ManualMaintenanceWindowConfig
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &cfg); err != nil {
				return nil, err
			}
		}
		return cfg, nil
	})

	return &monitorTypeMapper{
		toDBStatusRegistry:               toDBStatusMap,
		toDomainStatusRegistry:           toDomainStatusMap,
		toDBExpectationsRegistry:         toDBExpectationsMap,
		toDomainExpectationsRegistry:     toDomainExpectationsMap,
		toDomainTargetProtocolRegistry:   toDomainTargetProtocolMap,
		toDomainTargetConfigRegistry:     toDomainTargetConfigMap,
		toDomainContactTypeRegistry:      toDomainContactTypeMap,
		toDomainContactConfigRegistry:    toDomainContactConfigMap,
		toDomainWindowTypeRegistry:       toDomainWindowTypeMap,
		toDomainWindowConfigTypeRegistry: toDomainWindowConfigMap,
	}
}

func (m monitorTypeMapper) ToDBStatusType(status monitor.Status) (sqlcgen.StatusType, error) {
	return m.toDBStatusRegistry.Convert(status, status)
}

func (m monitorTypeMapper) ToDomainStatusType(status sqlcgen.StatusType) (monitor.Status, error) {
	return m.toDomainStatusRegistry.Convert(status, status)
}

func (m monitorTypeMapper) ToDBExpectations(expectations monitor.Expectations) ([]byte, error) {
	if expectations == nil {
		return nil, fmt.Errorf("expectations are required")
	}
	return m.toDBExpectationsRegistry.Convert(expectations.Protocol(), expectations)
}

func (m monitorTypeMapper) ToDomainExpectations(protocol target.Protocol, payload []byte) (monitor.Expectations, error) {
	return m.toDomainExpectationsRegistry.Convert(protocol, payload)
}

func (m monitorTypeMapper) ToDomainTargetProtocol(protocol sqlcgen.ProtocolType) (target.Protocol, error) {
	return m.toDomainTargetProtocolRegistry.Convert(protocol, protocol)
}

func (m monitorTypeMapper) ToDomainTargetNetworkConfig(protocol target.Protocol, payload []byte) (target.NetworkConfig, error) {
	return m.toDomainTargetConfigRegistry.Convert(protocol, payload)
}

func (m monitorTypeMapper) ToDomainContactType(contactType sqlcgen.ContactType) (alert.ContactType, error) {
	return m.toDomainContactTypeRegistry.Convert(contactType, contactType)
}

func (m monitorTypeMapper) ToDomainContactConfig(contactType alert.ContactType, payload []byte) (alert.ContactConfig, error) {
	return m.toDomainContactConfigRegistry.Convert(contactType, payload)
}

func (m monitorTypeMapper) ToDomainMaintenanceWindowType(windowType sqlcgen.MaintenanceType) (maintenance.WindowType, error) {
	return m.toDomainWindowTypeRegistry.Convert(windowType, windowType)
}

func (m monitorTypeMapper) ToDomainMaintenanceWindowConfig(windowType maintenance.WindowType, payload []byte) (maintenance.MaintenanceWindowConfig, error) {
	return m.toDomainWindowConfigTypeRegistry.Convert(windowType, payload)
}

func parseJSONArray(raw interface{}, dst interface{}) error {
	if raw == nil {
		return nil
	}

	var data []byte
	switch v := raw.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("unexpected JSON aggregate type: %T", raw)
	}

	if len(data) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, dst); err != nil {
		return err
	}

	return nil
}

