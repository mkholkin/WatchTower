package target

type Protocol string

const (
	ProtocolHTTP Protocol = "HTTP"
	ProtocolTCP  Protocol = "TCP"
	ProtocolICMP Protocol = "ICMP"
)

type NetworkConfig interface {
	Protocol() Protocol
	Validate() error
	isNetworkConfig()
}

// Marker interface to ensure that only valid network config types implement NetworkConfig.
type networkConfigMarker struct{}

func (networkConfigMarker) isNetworkConfig() {}

//TODO: возможно сокрыть реализации в будущем

// ----- HTTP -------

// HTTPConfig represents the configuration for an HTTP probe.
type HTTPConfig struct {
	networkConfigMarker
	Method          string            `json:"method"`
	Headers         map[string]string `json:"headers,omitempty"`
	Body            string            `json:"body,omitempty"`
	FollowRedirects bool              `json:"follow_redirects"`
}

func (c HTTPConfig) Protocol() Protocol {
	return ProtocolHTTP
}

func (c HTTPConfig) Validate() error {
	//TODO implement me
	panic("implement me")
}

// ----- TCP -------

// TCPConfig represents the configuration for a TCP probe.
type TCPConfig struct {
	networkConfigMarker
	Port int
}

func (c TCPConfig) Protocol() Protocol {
	return ProtocolTCP
}

func (c TCPConfig) Validate() error {
	//TODO implement me
	panic("implement me")
}

// ----- ICMP -------

// ICMPConfig represents the configuration for an ICMP (Ping) probe.
type ICMPConfig struct {
	networkConfigMarker
}

func (c ICMPConfig) Protocol() Protocol {
	return ProtocolICMP
}

func (c ICMPConfig) Validate() error {
	//TODO implement me
	panic("implement me")
}
