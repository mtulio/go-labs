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
			"service-tcp",
			op.ServicePort,
			&ctrl,
		)
		if err != nil {
			log.Fatal("ERROR creating Server Service", err)
		}
		ln.serverService = srvSvc

	case ProtoTLS:
		srvSvc, err := NewTLSServer(
			"service-tls",
			op.ServicePort,
			&ctrl,
			op.certPem,
			op.certKey,
		)
		if err != nil {
			log.Fatal("ERROR creating Server Service", err)
		}
		ln.serverService = srvSvc

	// case protoHTTP:
	// 	srvSvc, err := NewHTTPServer(
	// 		"service-http",
	// 		op.servicePort,
	// 		ec,
	// 		&ctrl,
	// 	)
	// 	if err != nil {
	// 		log.Fatal("ERROR creating Server Service", err)
	// 	}

	// case protoHTTPS:
	// 	srvSvc, err := NewHTTPSServer(
	// 		"service-https",
	// 		op.servicePort,
	// 		ec,
	// 		&ctrl,
	// 		op.certPem,
	// 		op.certKey,
	// 	)
	// 	if err != nil {
	// 		log.Fatal("ERROR creating Server Service", err)
	// 	}
	}

	// Create Server HC
	switch(op.HCProto) {
	case ProtoTCP:
		srvHC, err := NewTCPServer(
			"hc-tcp",
			op.HCPort,
			&ctrl,
		)
		if err != nil {
			log.Fatal("ERROR creating Server HC", err)
		}
		ln.serverHC = srvHC

	case ProtoTLS:
		srvHC, err := NewTLSServer(
			"hc-tls",
			op.HCPort,
			&ctrl,
			op.certPem,
			op.certKey,
		)
		if err != nil {
			log.Fatal("ERROR creating Server HC", err)
		}
		ln.serverHC = srvHC

	// case protoHTTP:
	// 	srvHC, err := NewHTTPServer(
	// 		"hc-http",
	// 		op.servicePort,
	// 		ec,
	// 		&ctrl,
	// 	)
	// 	if err != nil {
	// 		log.Fatal("ERROR creating Server HC", err)
	// 	}
	// case protoHTTPS:
	// 	srvHC, err := NewHTTPSServer(
	// 		"hc-https",
	// 		op.servicePort,
	// 		ec,
	// 		&ctrl,
	// 		op.certPem,
	// 		op.certKey,
	// 	)
	// 	if err != nil {
	// 		log.Fatal("ERROR creating Server HC", err)
	// 	}
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