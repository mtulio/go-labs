package main

import (
	"log"

	flag "github.com/spf13/pflag"

	"github.com/mtulio/go-lab-api/internal/server"
)

var (
	appName     *string = flag.String("app-name", "myApp", "help message for flagname")
	logPath     *string = flag.String("log-path", "", "help message for flagname")
	svcProto    *string = flag.String("service-proto", "http", "help message for flagname")
	svcPort     *uint64 = flag.Uint64("service-port", 30300, "help message for flagname")
	certPem     *string = flag.String("cert-pem", "", "help message for flagname")
	certKey     *string = flag.String("cert-key", "", "help message for flagname")
	hcProto     *string = flag.String("health-check-proto", "http", "help message for flagname")
	hcPort      *uint64 = flag.Uint64("health-check-port", 30301, "help message for flagname")
	hcPath      *string = flag.String("health-check-path", "/healthy", "help message for flagname")
	watchTg     *string = flag.String("watch-target-group-arn", "", "help message for flagname")
	termTimeout *int    = flag.Int("termination-timeout", 300, "help message for flagname")
)

func init() {
	// input
	// --service-proto --service-port
	// --health-check-proto --health-check-port --health-check-path
	// --watch-aws-tg-arn
	// --termination-timeout
	flag.Parse()
}

func main() {
	readyToShutdown := make(chan struct{})

	ev := server.NewEventHandler(*appName, *logPath)

	lnc := server.ListenerOptions{
		ServiceProto: server.GetProtocolByString(*svcProto),
		ServicePort:  *svcPort,
		HCProto:      server.GetProtocolByString(*hcProto),
		HCPort:       *hcPort,
		HCPath:       *hcPath,
		CertPem:      *certPem,
		CertKey:      *certKey,
	}

	ln, err := server.NewListener(&lnc, ev)
	if err != nil {
		log.Fatal("ERROR Creating the listener")
	}

	ln.Start()

	// read and print channel of requests
	<-readyToShutdown
}
