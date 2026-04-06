package mappers

import (
	"WatchTower/internal/domain/entity/monitor"
	"WatchTower/internal/service/monitoring_management/dto"
	"fmt"
)

// HTTPExpectationsMapper maps HTTP expectations payload to monitor.HTTPExpectations.
type HTTPExpectationsMapper struct{}

// Map converts typed expectations into domain HTTP expectations.
func (HTTPExpectationsMapper) Map(exp dto.MonitorExpectations) (monitor.Expectations, error) {
	var httpExp dto.HTTPMonitorExpectations
	switch typed := exp.(type) {
	case dto.HTTPMonitorExpectations:
		httpExp = typed
	case *dto.HTTPMonitorExpectations:
		if typed == nil {
			return nil, fmt.Errorf("http expectations are nil")
		}
		httpExp = *typed
	default:
		return nil, fmt.Errorf("invalid expectations type for HTTP protocol")
	}

	return monitor.HTTPExpectations{
		StatusCodes:  httpExp.StatusCodes,
		MaxLatencyMs: httpExp.MaxLatencyMs,
	}, nil
}

// TCPExpectationsMapper maps TCP expectations payload to monitor.TCPExpectations.
type TCPExpectationsMapper struct{}

// Map converts typed expectations into domain TCP expectations.
func (TCPExpectationsMapper) Map(exp dto.MonitorExpectations) (monitor.Expectations, error) {
	var tcpExp dto.TCPMonitorExpectations
	switch typed := exp.(type) {
	case dto.TCPMonitorExpectations:
		tcpExp = typed
	case *dto.TCPMonitorExpectations:
		if typed == nil {
			return nil, fmt.Errorf("tcp expectations are nil")
		}
		tcpExp = *typed
	default:
		return nil, fmt.Errorf("invalid expectations type for TCP protocol")
	}

	return monitor.TCPExpectations{MaxLatencyMs: tcpExp.MaxLatencyMs}, nil
}

// ICMPExpectationsMapper maps ICMP expectations payload to monitor.ICMPExpectations.
type ICMPExpectationsMapper struct{}

// Map converts typed expectations into domain ICMP expectations.
func (ICMPExpectationsMapper) Map(exp dto.MonitorExpectations) (monitor.Expectations, error) {
	var icmpExp dto.ICMPMonitorExpectations
	switch typed := exp.(type) {
	case dto.ICMPMonitorExpectations:
		icmpExp = typed
	case *dto.ICMPMonitorExpectations:
		if typed == nil {
			return nil, fmt.Errorf("icmp expectations are nil")
		}
		icmpExp = *typed
	default:
		return nil, fmt.Errorf("invalid expectations type for ICMP protocol")
	}

	return monitor.ICMPExpectations{
		MaxLatencyMs:         icmpExp.MaxLatencyMs,
		MaxPacketLossPercent: icmpExp.MaxPacketLossPercent,
	}, nil
}
