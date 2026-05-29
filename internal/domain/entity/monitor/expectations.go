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
	//TODO implement me
	panic("implement me")
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
