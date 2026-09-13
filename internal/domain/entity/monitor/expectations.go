package monitor

import (
	"WatchTower/internal/domain/entity/target"
)

type Expectations interface {
	Protocol() target.Protocol
	Validate() error
	isExpectations()
}

type expectationsMarker struct{}

func (expectationsMarker) isExpectations() {}

// ----- HTTP -------

// HTTPExpectations represents the expected outcomes for an HTTP probe.
type HTTPExpectations struct {
	expectationsMarker
	StatusCodes  []int `json:"status_code"`
	MaxLatencyMs int   `json:"max_latency_ms"`
}

func (e HTTPExpectations) Protocol() target.Protocol {
	return target.ProtocolHTTP
}

func (e HTTPExpectations) Validate() error {
	if e.MaxLatencyMs < 0 {
		return wrapValidation("maximum latency must not be negative")
	}
	if len(e.StatusCodes) == 0 {
		return wrapValidation("at least one HTTP status code is required")
	}
	for _, code := range e.StatusCodes {
		if code < 100 || code > 599 {
			return wrapValidation("HTTP status code must be between 100 and 599")
		}
	}
	return nil
}

// ----- TCP -------

// TCPExpectations represents the expected outcomes for a TCP probe.
type TCPExpectations struct {
	expectationsMarker
	MaxLatencyMs int `json:"max_latency_ms"`
}

func (e TCPExpectations) Protocol() target.Protocol {
	return target.ProtocolTCP
}

func (e TCPExpectations) Validate() error {
	//TODO implement me
	panic("implement me")
}

// ----- ICMP -------

// ICMPExpectations represents the expected outcomes for an ICMP (Ping) probe.
type ICMPExpectations struct {
	expectationsMarker
	MaxLatencyMs         int `json:"max_latency_ms"`
	MaxPacketLossPercent int `json:"max_packet_loss_percent"`
}

func (e ICMPExpectations) Protocol() target.Protocol {
	return target.ProtocolICMP
}

func (e ICMPExpectations) Validate() error {
	//TODO implement me
	panic("implement me")
}
