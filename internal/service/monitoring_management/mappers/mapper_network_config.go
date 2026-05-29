package mappers

import (
	"WatchTower/internal/domain/entity/target"
	"WatchTower/internal/service/monitoring_management/dto"
	"fmt"
)

// HTTPNetworkConfigMapper maps HTTP config payload to target.HTTPConfig.
type HTTPNetworkConfigMapper struct{}

// Map converts typed config into domain HTTP config.
func (HTTPNetworkConfigMapper) Map(cfg dto.MonitorNetworkConfig) (target.NetworkConfig, error) {
	var httpCfg dto.HTTPMonitorNetworkConfig
	switch typed := cfg.(type) {
	case dto.HTTPMonitorNetworkConfig:
		httpCfg = typed
	case *dto.HTTPMonitorNetworkConfig:
		if typed == nil {
			return nil, fmt.Errorf("http config is nil")
		}
		httpCfg = *typed
	default:
		return nil, fmt.Errorf("invalid config type for HTTP protocol")
	}

	return target.HTTPConfig{
		Method:          httpCfg.Method,
		Headers:         httpCfg.Headers,
		Body:            httpCfg.Body,
		FollowRedirects: httpCfg.FollowRedirects,
	}, nil
}

// TCPNetworkConfigMapper maps TCP config payload to target.TCPConfig.
type TCPNetworkConfigMapper struct{}

// Map converts typed config into domain TCP config.
func (TCPNetworkConfigMapper) Map(cfg dto.MonitorNetworkConfig) (target.NetworkConfig, error) {
	var tcpCfg dto.TCPMonitorNetworkConfig
	switch typed := cfg.(type) {
	case dto.TCPMonitorNetworkConfig:
		tcpCfg = typed
	case *dto.TCPMonitorNetworkConfig:
		if typed == nil {
			return nil, fmt.Errorf("tcp config is nil")
		}
		tcpCfg = *typed
	default:
		return nil, fmt.Errorf("invalid config type for TCP protocol")
	}

	return target.TCPConfig{Port: tcpCfg.Port}, nil
}

// ICMPNetworkConfigMapper maps ICMP config payload to target.ICMPConfig.
type ICMPNetworkConfigMapper struct{}

// Map converts typed config into domain ICMP config.
func (ICMPNetworkConfigMapper) Map(cfg dto.MonitorNetworkConfig) (target.NetworkConfig, error) {
	switch typed := cfg.(type) {
	case dto.ICMPMonitorNetworkConfig:
		_ = typed
	case *dto.ICMPMonitorNetworkConfig:
		if typed == nil {
			return nil, fmt.Errorf("icmp config is nil")
		}
	default:
		return nil, fmt.Errorf("invalid config type for ICMP protocol")
	}

	return target.ICMPConfig{}, nil
}
