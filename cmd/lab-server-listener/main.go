package main;

import (
	"log"
	"github.com/mtulio/go-lab-api/internal/server"
)

func init() {
	// input
	// --service-proto --service-port 
	// --health-check-proto --health-check-port --health-check-path
	// --watch-aws-tg-arn
	// --termination-timeout
}

func main() {
	readyToShutdown := make(chan struct{})
	
	lnc := server.ListenerOptions{
		ServiceProto: server.ProtoHTTP,
		ServicePort: 30300,
		HCProto: server.ProtoHTTP,
		HCPort: 30301,
	}

	ln, err := server.NewListener(&lnc)
	if err != nil {
		log.Fatal("ERROR Creating the listener")
	}

	ln.Start()

	// read and print channel of requests
	<-readyToShutdown
}