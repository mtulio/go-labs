package server

import (
	"log"

	"github.com/mtulio/go-lab-api/internal/event"
	"github.com/mtulio/go-lab-api/internal/metric"
)

type ListenerOptions struct {
	ServiceProto       Protocol
	ServicePort        uint64
	HCProto            Protocol
	HCPort             uint64
	HCPath             string
	TargetGroupARN     string
	CertPem            string
	CertKey            string
	TerminationTimeout float64
	Event              *event.EventHandler
	Metric             *metric.MetricsHandler
}

type Listener struct {
	options       *ListenerOptions
	serverService Server
	serverHC      Server
	controllerHC  *HealthCheckController
	Event         *event.EventHandler
	watcherTG     *TargetGroupWatcher
}

func NewListener(op *ListenerOptions) (*Listener, error) {

	// Create HC Controller
	ctrl := NewHealthCheckController(&HCControllerOpts{
		Event:  op.Event,
		Metric: op.Metric,
	})
	watcher, err := NewWatcherTargetGroup(&TGWatcherOptions{
		ARN:    op.TargetGroupARN,
		Metric: op.Metric,
	})
	if err != nil {
		log.Fatal(err)
	}
	//watcher.Start()

	ln := Listener{
		options:      op,
		controllerHC: ctrl,
		Event:        op.Event,
		watcherTG:    watcher,
	}

	switch op.ServiceProto {
	case ProtoTCP:
		srvSvc, err := NewTCPServer(&ServerConfig{
			name:     "server-service-tcp",
			proto:    ProtoTCP,
			port:     op.ServicePort,
			hcServer: false,
			hc:       ctrl,
			event:    op.Event,
			metric:   op.Metric,
		})
		if err != nil {
			log.Fatal("ERROR creating Server Service", err)
		}
		ln.serverService = srvSvc

	case ProtoTLS:
		srvSvc, err := NewTCPServer(&ServerConfig{
			name:     "server-service-tls",
			proto:    ProtoTLS,
			port:     op.ServicePort,
			hcServer: false,
			hc:       ctrl,
			event:    op.Event,
			metric:   op.Metric,
			certPem:  op.CertPem,
			certKey:  op.CertKey,
		})
		if err != nil {
			log.Fatal("ERROR creating Server Service", err)
		}
		ln.serverService = srvSvc

	case ProtoHTTP:
		srvSvc, err := NewHTTPServer(&ServerConfig{
			name:     "server-service-http",
			proto:    ProtoHTTP,
			port:     op.ServicePort,
			hcServer: true,
			hc:       ctrl,
			event:    op.Event,
			metric:   op.Metric,
			certPem:  op.CertPem,
			certKey:  op.CertKey,
		})
		if err != nil {
			log.Fatal("ERROR creating Server Service", err)
		}
		ln.serverService = srvSvc

	case ProtoHTTPS:
		srvSvc, err := NewHTTPServer(&ServerConfig{
			name:     "server-service-https",
			proto:    ProtoHTTPS,
			port:     op.ServicePort,
			hcServer: true,
			hc:       ctrl,
			event:    op.Event,
			metric:   op.Metric,
			certPem:  op.CertPem,
			certKey:  op.CertKey,
		})
		if err != nil {
			log.Fatal("ERROR creating Server Service", err)
		}
		ln.serverService = srvSvc
	}

	// Create Server HC
	switch op.HCProto {
	case ProtoTCP:
		srvHC, err := NewTCPServer(&ServerConfig{
			name:     "server-hc-tcp",
			proto:    ProtoTCP,
			port:     op.HCPort,
			hcServer: true,
			hc:       ctrl,
			event:    op.Event,
			metric:   op.Metric,
		})
		if err != nil {
			log.Fatal("ERROR creating Server HC", err)
		}
		ln.serverHC = srvHC

	case ProtoTLS:
		srvHC, err := NewTCPServer(&ServerConfig{
			name:     "server-hc-tls",
			proto:    ProtoTLS,
			port:     op.HCPort,
			hcServer: true,
			hc:       ctrl,
			event:    op.Event,
			metric:   op.Metric,
			certPem:  op.CertPem,
			certKey:  op.CertKey,
		})
		if err != nil {
			log.Fatal("ERROR creating Server HC", err)
		}
		ln.serverHC = srvHC

	case ProtoHTTP:
		srvHC, err := NewHTTPServer(&ServerConfig{
			name:     "server-hc-http",
			proto:    ProtoHTTP,
			port:     op.HCPort,
			hcServer: true,
			hc:       ctrl,
			event:    op.Event,
			metric:   op.Metric,
		})
		if err != nil {
			log.Fatal("ERROR creating Server HC", err)
		}
		ln.serverHC = srvHC

	case ProtoHTTPS:
		srvHC, err := NewHTTPServer(&ServerConfig{
			name:     "server-hc-https",
			proto:    ProtoHTTPS,
			port:     op.HCPort,
			hcServer: true,
			hc:       ctrl,
			event:    op.Event,
			metric:   op.Metric,
			certPem:  op.CertPem,
			certKey:  op.CertKey,
		})
		if err != nil {
			log.Fatal("ERROR creating Server HC", err)
		}
		ln.serverHC = srvHC
	}

	return &ln, nil
}

func (l *Listener) Start() error {
	l.Event.Send("runtime", "listener", "Starting services...")

	// Start LoadBalancer/TargetGroup watcher
	go l.watcherTG.Start()

	// Start HC Controller
	go l.controllerHC.Start()

	// Start HC server
	go l.serverHC.Start()

	// Start Service server
	go l.serverService.Start()

	return nil
}
