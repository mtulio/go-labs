/*
Kubernetes apiserver cluster healthy watcher

This tools will observe:
- all KAS /healthy endpoints (kube-apiservers for each master)
- NLB's TG targets and it's status
- Make requests to the NLB's public endpoints /healthy and register failed requests

*/
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	flag "github.com/spf13/pflag"

	"github.com/mtulio/go-lab-api/internal/event"
	"github.com/mtulio/go-lab-api/internal/metric"
	"github.com/mtulio/go-lab-api/internal/utils"
	"github.com/mtulio/go-lab-api/internal/watcher"
)

type Watcher struct {
	name    string
	intURLs []string
	extURL  string
	wg      sync.WaitGroup
	e       *event.EventHandler
	m       *metric.MetricsHandler
	awsTgW  *watcher.TargetGroupWatcher
	finish  bool
}

var (
	logPath       *string = flag.String("log-path", "", "help message for flagname")
	urlTargets    *string = flag.String("url-targets", "", "List of internal URLs of targets to watch / make requests.  Example: https://10.5.5.5:port/path,https://10.5.5.5:6443/readyz")
	urlLb         *string = flag.String("url-lb", "", "Public URL Load Balancer to monitor / make requests. Example: https://LB_DNS:port/path")
	lbProvider    *string = flag.String("lb-provider", "", "AWS NLB Target Group ARN to watch")
	lbId          *string = flag.String("lb-id", "", "Load Balancer ID. For AWS NLB Target Group ARN to watch")
	monitMode     *string = flag.String("mode", "sync", "Monitor mode: sync|async. Sync will monitor on Prometheus scrape. Async will start a thread to monit it in --interval.")
	monitInterval *int    = flag.Int("interval", 5, "Interval in seconds to monitor resources. Default: 5")
	w             *Watcher
)

func init() {
	flag.Parse()
	if *lbId == "" {
		fmt.Println("Target Group ARN must be set: --target-group-arn")
		os.Exit(1)
	}
	if *urlTargets == "" {
		fmt.Println("Kube-apiserver endpoints must be set. Example: https://10.0.0.1:6443/readyz,https://10.0.0.2:6443/readyz")
		os.Exit(1)
	}
	if *urlLb == "" {
		fmt.Println("NLB Public endpoint must be set. Example: https://nlb-public-dns.elb.us-east-1.amazonaws.com:6443/readyz")
		os.Exit(1)
	}
	// split internal urls
	wname := "lb-watcher"
	w = &Watcher{
		name:    wname,
		intURLs: strings.Split(*urlTargets, ","),
		extURL:  *urlLb,
		e:       event.NewEventHandler(wname, *logPath),
		finish:  false,
	}
	w.m = metric.NewMetricHandler(w.e)
}

func main() {
	// Start metrics dumper/pusher
	if *monitMode == "async" {
		go w.m.StartPusher()
	}

	if *lbId != "" {
		tgw, err := watcher.NewTargetGroupWatcher(&watcher.TGWatcherOptions{
			ARN:    *lbId,
			Metric: w.m,
		})
		if err != nil {
			log.Fatal(err)
		}
		w.awsTgW = tgw
		if *monitMode == "async" {
			go tgw.Start()
		}
	}

	serverPort := ":9999"
	exporterPath := "/metrics"
	exporter := NewWatcherCollector()
	prometheus.MustRegister(exporter)

	log.Printf("Starting http server on port %s", serverPort)
	http.Handle(exporterPath, promhttp.Handler())
	log.Fatal(http.ListenAndServe(serverPort, nil))
}

type watcherCollector struct {
	urlReqSuccessMetric    *prometheus.Desc
	urlReqTotalMetric      *prometheus.CounterVec
	lbTargetHealthMetric   *prometheus.Desc
	lbTargetReqTotalMetric *prometheus.CounterVec
}

func NewWatcherCollector() *watcherCollector {
	w := &watcherCollector{
		urlReqSuccessMetric: prometheus.NewDesc("url_request_success",
			"Boolean metric indicating that the request was finisied with success",
			[]string{"target", "type"}, nil,
		),
		// urlReqCountMetric: prometheus.NewDesc("url_request_count",
		// 	"Request counter",
		// 	[]string{"target", "type"}, nil,
		// ),
		urlReqTotalMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "url_requests_total",
				Help: "Number of requests to endpoints.",
			},
			[]string{"target", "type"},
		),
		lbTargetHealthMetric: prometheus.NewDesc("lb_target_healthy",
			"Boolean indicating if the target is healthy on LB",
			[]string{"target"}, nil,
		),
		lbTargetReqTotalMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "lb_target_requests_total",
				Help: "Number of requests to discovery LB targets.",
			},
			[]string{"type"},
		),
	}
	prometheus.MustRegister(w.urlReqTotalMetric)
	prometheus.MustRegister(w.lbTargetReqTotalMetric)
	return w
}

//Describe essentially writes all descriptors to the prometheus desc channel.
func (collector *watcherCollector) Describe(ch chan<- *prometheus.Desc) {

	ch <- collector.urlReqSuccessMetric
	ch <- collector.lbTargetHealthMetric
}

//Collect implements required collect function for all promehteus collectors
func (collector *watcherCollector) Collect(ch chan<- prometheus.Metric) {
	wname := "lb-watcher-collector"
	wc := &Watcher{
		name:    wname,
		intURLs: strings.Split(*urlTargets, ","),
		extURL:  *urlLb,
		finish:  false,
	}

	// Collecting external urls
	for _, url := range wc.intURLs {
		wc.wg.Add(1)
		go func(u string) {
			defer wc.wg.Done()
			ch <- prometheus.MustNewConstMetric(
				collector.urlReqSuccessMetric,
				prometheus.CounterValue,
				utils.CheckRequestRespHealthyMetric(utils.MakeRequest(w.e, u)),
				u, "target",
			)
			collector.urlReqTotalMetric.WithLabelValues(u, "target").Inc()
		}(url)
	}

	// Collect external endpoint
	wc.wg.Add(1)
	go func(u string) {
		defer wc.wg.Done()
		ch <- prometheus.MustNewConstMetric(
			collector.urlReqSuccessMetric,
			prometheus.CounterValue,
			utils.CheckRequestRespHealthyMetric(utils.MakeRequest(w.e, u)),
			u, "lb",
		)
		collector.urlReqTotalMetric.WithLabelValues(u, "lb").Inc()
	}(wc.extURL)

	// collect TG healthy info
	wc.wg.Add(1)
	go func() {
		defer wc.wg.Done()
		respTg := w.awsTgW.Collect()
		for k, v := range *respTg {
			ch <- prometheus.MustNewConstMetric(
				collector.lbTargetHealthMetric,
				prometheus.CounterValue,
				float64(v),
				k,
			)
		}
		collector.lbTargetReqTotalMetric.WithLabelValues("lb").Inc()
	}()

	wc.wg.Wait()
}
