package server

import (
	"github.com/mtulio/go-lab-api/internal/event"
	"github.com/mtulio/go-lab-api/internal/metric"
)

type Protocol uint8

const (
	ProtoTCP Protocol = iota
	ProtoTLS
	ProtoHTTP
	ProtoHTTPS
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
	listener *Listener
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
