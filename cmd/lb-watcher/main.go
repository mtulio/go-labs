/*
Kubernetes apiserver cluster healthy watcher

This tools will observe:
- all KAS /healthy endpoints (kube-apiservers for each master)
- NLB's TG targets and it's status
- Make requests to the NLB's public endpoints /healthy and register failed requests

*/
package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	flag "github.com/spf13/pflag"

	"github.com/mtulio/go-lab-api/internal/event"
	"github.com/mtulio/go-lab-api/internal/metric"
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
	logPath      *string = flag.String("log-path", "", "help message for flagname")
	urlsInt      *string = flag.String("url-targets", "", "List of internal URLs of targets to watch / make requests.  Example: https://10.5.5.5:port/path,https://10.5.5.5:6443/readyz")
	urlExt       *string = flag.String("url-lb", "", "Public URL Load Balancer to monitor / make requests. Example: https://LB_DNS:port/path")
	watchLBAwsTg *string = flag.String("lb-aws-target-group-arn", "", "AWS NLB Target Group ARN to watch")
	w            *Watcher
)

func init() {
	flag.Parse()
	if *watchLBAwsTg == "" {
		fmt.Println("Target Group ARN must be set: --target-group-arn")
		os.Exit(1)
	}
	if *urlsInt == "" {
		fmt.Println("Kube-apiserver endpoints must be set. Example: https://10.0.0.1:6443/readyz,https://10.0.0.2:6443/readyz")
		os.Exit(1)
	}
	if *urlExt == "" {
		fmt.Println("NLB Public endpoint must be set. Example: https://nlb-public-dns.elb.us-east-1.amazonaws.com:6443/readyz")
		os.Exit(1)
	}
	// split internal urls
	wname := "lb-watcher"
	w = &Watcher{
		name:    wname,
		intURLs: strings.Split(*urlsInt, ","),
		extURL:  *urlExt,
		e:       event.NewEventHandler(wname, *logPath),
		finish:  false,
	}
	w.m = metric.NewMetricHandler(w.e)
}

func main() {
	// Start metrics dumper/pusher
	go w.m.StartPusher()

	// start watching target group to extract metrics
	// dry-run / run locally
	if *watchLBAwsTg != "" {
		tgw, err := watcher.NewTargetGroupWatcher(&watcher.TGWatcherOptions{
			ARN:    *watchLBAwsTg,
			Metric: w.m,
		})
		if err != nil {
			log.Fatal(err)
		}
		w.awsTgW = tgw
		go tgw.Start()
	}

	// start apiserver client requests
	//> start
	// initWatcherInternalURLs(w)
	// initWatcherExternalURL(w)
	// w.wg.Wait()

	exporter := NewWatcherCollector()
	prometheus.MustRegister(exporter)

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":9999", nil))
}

func initWatcherInternalURLs(w *Watcher) {
	for _, url := range w.intURLs {
		w.wg.Add(1)
		go func(u string) {
			defer w.wg.Done()
			for {
				healthy := makeRequest(w.e, u)
				log.Println(u, healthy)
				//m.Inc("requests_hc")
				time.Sleep(1 * time.Second)
				if w.finish {
					return
				}
			}
		}(url)
	}
}

func initWatcherExternalURL(w *Watcher) {
	w.wg.Add(1)
	go func(u string) {
		defer w.wg.Done()
		for {
			healthy := makeRequest(w.e, u)
			log.Println(u, healthy)
			//m.Inc("requests_hc")
			time.Sleep(1 * time.Second)
			if w.finish {
				return
			}
		}
	}(w.extURL)
}

func makeRequest(e *event.EventHandler, url string) bool {
	tlsCfg := tls.Config{InsecureSkipVerify: true}
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tlsCfg
	client := http.Client{
		Timeout: 1 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		msg := fmt.Sprintf("ERROR [%s] received from server. Delaying 10s: %s", url, err)
		e.Send("request-client", url, msg)
		return false
	}

	return (resp.StatusCode >= 200 && resp.StatusCode < 400)
}

type watcherCollector struct {
	urlReqSuccessMetric  *prometheus.Desc
	urlReqTotalMetric    *prometheus.CounterVec
	lbTargetHealthMetric *prometheus.Desc
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
				Name: "url_request_total",
				Help: "Number of requests to endpoints.",
			},
			[]string{"target", "type"},
		),
		lbTargetHealthMetric: prometheus.NewDesc("lb_target_healthy",
			"Boolean indicating if the target is healthy on LB",
			[]string{"target"}, nil,
		),
	}
	prometheus.MustRegister(w.urlReqTotalMetric)
	return w
}

//Each and every collector must implement the Describe function.
//It essentially writes all descriptors to the prometheus desc channel.
func (collector *watcherCollector) Describe(ch chan<- *prometheus.Desc) {

	//Update this section with the each metric you create for a given collector
	ch <- collector.urlReqSuccessMetric
	// ch <- collector.urlReqTotalMetric
	ch <- collector.lbTargetHealthMetric
}

//Collect implements required collect function for all promehteus collectors
func (collector *watcherCollector) Collect(ch chan<- prometheus.Metric) {
	wname := "lb-watcher-collector"
	wc := &Watcher{
		name:    wname,
		intURLs: strings.Split(*urlsInt, ","),
		extURL:  *urlExt,
		finish:  false,
	}

	// Collecting external urls
	for _, url := range wc.intURLs {
		wc.wg.Add(1)
		go func(u string) {
			defer wc.wg.Done()
			healthy := makeRequest(w.e, u)
			//log.Println(u, healthy)
			//time.Sleep(1 * time.Second)
			var vHealthy float64 = 0
			if healthy {
				vHealthy = 1
			}
			ch <- prometheus.MustNewConstMetric(
				collector.urlReqSuccessMetric,
				prometheus.CounterValue,
				vHealthy,
				u, "target",
			)
			collector.urlReqTotalMetric.WithLabelValues(u, "target").Inc()
		}(url)
	}

	// Collect external endpoint
	wc.wg.Add(1)
	go func(u string) {
		defer wc.wg.Done()
		healthy := makeRequest(w.e, u)
		//log.Println(u, healthy)
		//m.Inc("requests_hc")
		var vHealthy float64 = 0
		if healthy {
			vHealthy = 1
		}
		ch <- prometheus.MustNewConstMetric(
			collector.urlReqSuccessMetric,
			prometheus.CounterValue,
			vHealthy,
			u, "lb",
		)
		collector.urlReqTotalMetric.WithLabelValues(u, "target").Inc()
	}(wc.extURL)

	// collect TG healthy info
	wc.wg.Add(1)
	go func() {
		defer wc.wg.Done()
		respTg := w.awsTgW.Collect()
		log.Println(respTg)
		for k, v := range *respTg {
			log.Println(k)
			ch <- prometheus.MustNewConstMetric(
				collector.urlReqSuccessMetric,
				prometheus.CounterValue,
				float64(v),
				k, "lb",
			)
		}
	}()

	wc.wg.Wait()
}
