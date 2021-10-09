package server;

import (
	"log"
	"fmt"
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
	serverService   *ServerTCP
	serverHC        *ServerTCP
	//watcher       *TargetGroupWatcher
	controllerHC    *HealthCheckController 
	events          *chan string
}

func NewListener(ec *chan string, op *ListenerOptions) (*Listener, error) {

	// Create HC Controller
	ctrl := HealthCheckController{}

	// Create Server Service
	var srvSvc *ServerTCP
	var srvHC *ServerTCP
	var err error
	switch(op.ServiceProto) {
	case ProtoTCP:
		srvSvc, err = NewTCPServer(
			"service-tcp",
			op.ServicePort,
			ec,
			&ctrl,
		)
		if err != nil {
			log.Fatal("ERROR creating Server Service", err)
		}
	// case ProtoTLS:
	// 	srvSvc, err = NewTLSServer(
	// 		"service-tls",
	// 		op.ServicePort,
	// 		ec,
	// 		&ctrl,
	// 		op.certPem,
	// 		op.certKey,
	// 	)
	// 	if err != nil {
	// 		log.Fatal("ERROR creating Server Service", err)
	// 	}
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
		srvHC, err = NewTCPServer(
			"hc-tcp",
			op.HCPort,
			ec,
			&ctrl,
		)
		if err != nil {
			log.Fatal("ERROR creating Server HC", err)
		}
	// case ProtoTLS:
	// 	srvHC, err = NewTLSServer(
	// 		"hc-tls",
	// 		op.HCPort,
	// 		ec,
	// 		&ctrl,
	// 		op.certPem,
	// 		op.certKey,
	// 	)
	// 	if err != nil {
	// 		log.Fatal("ERROR creating Server HC", err)
	// 	}
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

	// Create TG Watcher
	//ToDo

	return &Listener{
		options: op,
		events: ec,
		serverService: srvSvc,
		serverHC: srvHC,
		controllerHC: &ctrl,
	}, nil
}

func (l *Listener) Start() (error) {
	// Start watch target

	// Star HC Controller
	
	// Start HC server
	fmt.Println("Starting.....")
	go l.serverHC.Start()

	// Start Service server
	go l.serverService.Start()

	return nil
}