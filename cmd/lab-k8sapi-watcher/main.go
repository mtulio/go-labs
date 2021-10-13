package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/mtulio/go-lab-api/internal/server"
)

var (
	watchTg  *string = flag.String("target-group-arn", "", "Target Group ARN")
	endpoint *string = flag.String("endpoint", "https://localhost:6443/readyz", "k8s-api healthy endpoint")
	logPath  *string = flag.String("log-path", "", "help message for flagname")
)

func init() {
	flag.Parse()
	if *endpoint == "" {
		fmt.Println("Target Group ARN must be set: --target-group-arn")
		os.Exit(1)
	}
}

// Start register when termination start. Should be called when
// the signal is sent to k8s-apiserver.
func signalHandler(m *server.MetricsHandler, e *server.EventHandler) {
	for {
		msg := ("Running Signal handler")
		e.Send("runtime", "hc-controller", msg)

		termChan := make(chan os.Signal)
		signal.Notify(termChan, syscall.SIGTERM)

		<-termChan

		msg = ("Termination Signal receievd")
		e.Send("runtime", "k8s-watcher-signal", msg)

		m.AppTermination = true

		termChan = make(chan os.Signal)
		signal.Notify(termChan, syscall.SIGTERM)
	}
}

func main() {
	appName := "k8sapi-watcher"
	e := server.NewEventHandler(appName, *logPath)
	m := server.NewMetricHandler(e)

	// set defaults
	m.AppHealthy = false
	m.AppTermination = false
	m.TargetHealthy = false

	// start signal handler
	go signalHandler(m, e)

	// Start metrics dumper/pusher
	go m.StartPusher()

	// start watching listener
	// dry-run / run locally
	if *watchTg != "" {
		tgw, err := server.NewWatcherTargetGroup(&server.TGWatcherOptions{
			ARN:    *watchTg,
			Metric: m,
			Event:  e,
		})
		if err != nil {
			log.Fatal(e)
		}
		go tgw.Start()
	}

	// start apiserver client requests
	//> slow start
	e.Send("request-client", appName, "Starting client in 10s")
	time.Sleep(10 * time.Second)
	for {
		tlsCfg := tls.Config{InsecureSkipVerify: true}
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tlsCfg

		resp, err := http.Get(*endpoint)
		if err != nil {
			msg := fmt.Sprintf("ERROR received from server. Delaying 10s: %s", err)
			e.Send("request-client", appName, msg)
			time.Sleep(10 * time.Second)
			continue
		}

		m.AppHealthy = (resp.StatusCode >= 200 && resp.StatusCode < 400)

		// make sure that termination flag will be clear when termination
		// was in progress and the App is operational
		if (m.AppTermination == true) && (m.AppHealthy == true) {
			m.AppTermination = !(m.AppHealthy)
		}
		m.Inc("requests_hc")
		time.Sleep(1 * time.Second)
	}
}
