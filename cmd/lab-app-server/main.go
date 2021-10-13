package main

import (
	"log"

	flag "github.com/spf13/pflag"

	"github.com/mtulio/go-lab-api/internal/event"
	"github.com/mtulio/go-lab-api/internal/metric"
	"github.com/mtulio/go-lab-api/internal/server"
	"github.com/mtulio/go-lab-api/internal/watcher"
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
	hcPath      *string = flag.String("health-check-path", "/readyz", "help message for flagname")
	watchTg     *string = flag.String("watch-target-group-arn", "", "help message for flagname")
	termTimeout *int    = flag.Int("termination-timeout", 300, "help message for flagname")
)

func main() {
	flag.Parse()
	readyToShutdown := make(chan struct{})

	ev := event.NewEventHandler(*appName, *logPath)
	metric := metric.NewMetricHandler(ev)
	go metric.StartPusher()

	// Watch Target Group and extract/update metrics
	tgw, err := watcher.NewTargetGroupWatcher(&watcher.TGWatcherOptions{
		ARN:    *watchTg,
		Metric: metric,
	})
	if err != nil {
		log.Fatal(err)
	}
	go tgw.Start()

	// the listener will handle the servers (service and health-check)
	lnc := server.ListenerOptions{
		ServiceProto: server.GetProtocolFromStr(*svcProto),
		ServicePort:  *svcPort,
		HCProto:      server.GetProtocolFromStr(*hcProto),
		HCPort:       *hcPort,
		HCPath:       *hcPath,
		CertPem:      *certPem,
		CertKey:      *certKey,
		Event:        ev,
		Metric:       metric,
	}

	ln, err := server.NewListener(&lnc)
	if err != nil {
		log.Fatal("ERROR Creating the listener")
	}

	ln.Start()

	<-readyToShutdown
}
