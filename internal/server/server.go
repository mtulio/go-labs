package server

type Protocol uint8

const (
	ProtoTCP Protocol = iota
	ProtoTLS
	ProtoHTTP
	ProtoHTTPS
)

type Server interface {
	Start()
	//ShutdownHealthy() error
	//GetType() string
	//GetState() bool
	//SetState(value bool) error
}

func GetProtocolByString(proto string) Protocol {
	switch proto {
	case "tcp":
		return ProtoTCP
	case "tls":
		return ProtoTLS
	case "http":
		return ProtoHTTP
	case "https":
		return ProtoHTTPS
	}
	return ProtoHTTP
}
