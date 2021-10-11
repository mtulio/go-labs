package server;

import (
	"log"
)

type ListenerOptions struct {
	ServiceProto Protocol
	ServicePort uint64
	HCProto Protocol
	HCPort uint64
	hcPath string
	targetGroupARN string
	certPem string
	certKey string
}

type Listener struct {
	options         *ListenerOptions
	serverService   Server
	serverHC        Server
	//watcher       *TargetGroupWatcher
	controllerHC    *HealthCheckController 
	//events          *chan string
}

func NewListener( op *ListenerOptions) (*Listener, error) {

	// Create HC Controller
	ctrl := HealthCheckController{
		Healthy: true,
	}

	ln := Listener{
		options: op,
		controllerHC: &ctrl,
	}

	switch(op.ServiceProto) {
	case ProtoTCP:
		srvSvc, err := NewTCPServer(
			"server-service-tcp",
			op.ServicePort,
			&ctrl,
			false,
		)
		if err != nil {
			log.Fatal("ERROR creating Server Service", err)
		}
		ln.serverService = srvSvc

	case ProtoTLS:
		srvSvc, err := NewTLSServer(
			"server-service-tls",
			op.ServicePort,
			&ctrl,
			false,
			op.certPem,
			op.certKey,
		)
		if err != nil {
			log.Fatal("ERROR creating Server Service", err)
		}
		ln.serverService = srvSvc

	case ProtoHTTP:
		srvSvc, err := NewHTTPServer(
			"server-service-http",
			op.ServicePort,
			&ctrl,
			false,
		)
		if err != nil {
			log.Fatal("ERROR creating Server Service", err)
		}
		ln.serverService = srvSvc

	case ProtoHTTPS:
		srvSvc, err := NewHTTPSServer(
			"server-service-https",
			op.ServicePort,
			&ctrl,
			false,
			op.certPem,
			op.certKey,
		)
		if err != nil {
			log.Fatal("ERROR creating Server Service", err)
		}
		ln.serverService = srvSvc
	}

	// Create Server HC
	switch(op.HCProto) {
	case ProtoTCP:
		srvHC, err := NewTCPServer(
			"server-hc-tcp",
			op.HCPort,
			&ctrl,
			true,
		)
		if err != nil {
			log.Fatal("ERROR creating Server HC", err)
		}
		ln.serverHC = srvHC

	case ProtoTLS:
		srvHC, err := NewTLSServer(
			"server-hc-tls",
			op.HCPort,
			&ctrl,
			true,
			op.certPem,
			op.certKey,
		)
		if err != nil {
			log.Fatal("ERROR creating Server HC", err)
		}
		ln.serverHC = srvHC

	case ProtoHTTP:
		srvHC, err := NewHTTPServer(
			"server-hc-http",
			op.HCPort,
			&ctrl,
			true,
		)
		if err != nil {
			log.Fatal("ERROR creating Server HC", err)
		}
		ln.serverHC = srvHC
	case ProtoHTTPS:
		srvHC, err := NewHTTPSServer(
			"server-hc-https",
			op.HCPort,
			&ctrl,
			true,
			op.certPem,
			op.certKey,
		)
		if err != nil {
			log.Fatal("ERROR creating Server HC", err)
		}
		ln.serverHC = srvHC
	}

	return &ln, nil
}

func (l *Listener) Start() (error) {
	SendEvent("runtime", "listener", "Starting services...")

	// Start LoadBalancer/TargetGroup watcher
	//ToDo

	// Start HC Controller
	go l.controllerHC.Start()

	// Start HC server
	go l.serverHC.Start()

	// Start Service server
	go l.serverService.Start()

	return nil
}