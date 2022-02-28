package server

import (
	"github.com/mtulio/go-lab-api/internal/event"
	"github.com/mtulio/go-lab-api/internal/metric"
)

type Protocol uint8

const (
	ProtoTCP      Protocol = iota
	ProtoTLS      Protocol = iota
	ProtoHTTP     Protocol = iota
	ProtoHTTPS    Protocol = iota
	ServerTypeHC  uint8    = iota
	ServerTypeSvc uint8    = iota
)

type Server interface {
	Start()
	StartController()
	//ShutdownHealthy() error
	//GetType() string
	//GetState() bool
	//SetState(value bool) error
}

type ServerConfig struct {
	proto    Protocol
	name     string
	port     uint64
	event    *event.EventHandler
	metric   *metric.MetricsHandler
	hc       *HealthCheckController
	hcServer bool
	hcPath   string
	certPem  string
	certKey  string
	debug    bool
	sType    uint8
}

func GetProtocolFromStr(proto string) Protocol {
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

func GetServerTypeFromStr(t string) uint8 {
	switch t {
	case "healthcheck":
		return ServerTypeHC
	case "hc":
		return ServerTypeHC
	case "service":
		return ServerTypeSvc
	case "svc":
		return ServerTypeSvc
	default:
		return ServerTypeSvc
	}
}

func GetServerTypeToStr(t uint8) string {
	switch t {
	case ServerTypeHC:
		return "healthcheck"
	case ServerTypeSvc:
		return "service"
	default:
		return "service"
	}
}
