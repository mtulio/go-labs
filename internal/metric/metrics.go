package metric

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/mtulio/go-lab-api/internal/event"
	"github.com/prometheus/client_golang/prometheus"
)

type PromCollector struct {
	httpReqTotal *prometheus.CounterVec
}

type MetricsHandler struct {
	Time time.Time `json:"time"`

	// Global metrics
	mxGlobal            sync.Mutex
	AppTermination      bool   `json:"app_termination"`
	AppHealthy          bool   `json:"app_healthy"`
	TargetHealthy       bool   `json:"tg_healthy"`
	TargetHealthCount   uint64 `json:"tg_health_count"`
	TargetUnhealthCount uint64 `json:"tg_unhealth_count"`

	// Request counters
	mxReqService      sync.Mutex
	ReqCountService   uint64 `json:"reqc_service"`
	mxReqHC           sync.Mutex
	ReqCountHC        uint64 `json:"reqc_hc"`
	mxReqCli          sync.Mutex
	ReqCountClient    uint64 `json:"reqc_client"`
	ReqCountClient2xx uint64 `json:"reqc_client_2xx"`
	ReqCountClient4xx uint64 `json:"reqc_client_4xx"`
	ReqCountClient5xx uint64 `json:"reqc_client_5xx"`

	event *event.EventHandler

	PromExporter *PromCollector
}

func NewPromCollector() *PromCollector {
	c := &PromCollector{
		httpReqTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "app_handler_requests_total",
				Help: "Number of requests to a handler.",
			},
			[]string{"code", "type", "path"},
		),
	}
	prometheus.MustRegister(c.httpReqTotal)
	return c
}

func NewMetricHandler(e *event.EventHandler) *MetricsHandler {
	return &MetricsHandler{
		event:               e,
		AppTermination:      false,
		AppHealthy:          false,
		TargetHealthy:       false,
		TargetHealthCount:   0,
		TargetUnhealthCount: 0,
		ReqCountService:     0,
		ReqCountHC:          0,
		ReqCountClient:      0,
		PromExporter:        NewPromCollector(),
	}
}

func (m *MetricsHandler) SetCounter(metric string, value uint8) error {
	return nil
}

func (m *MetricsHandler) Inc(metric string) {
	switch metric {
	case "requests_service":
		m.mxReqService.Lock()
		m.ReqCountService += 1
		m.mxReqService.Unlock()
	case "requests_hc":
		m.mxReqService.Lock()
		m.ReqCountHC += 1
		m.mxReqService.Unlock()
	case "requests_client":
		m.mxReqCli.Lock()
		m.ReqCountClient += 1
		m.mxReqCli.Unlock()
	case "requests_cli_2xx":
		m.mxReqCli.Lock()
		m.ReqCountClient2xx += 1
		m.mxReqCli.Unlock()
	case "requests_cli_4xx":
		m.mxReqCli.Lock()
		m.ReqCountClient4xx += 1
		m.mxReqCli.Unlock()
	case "requests_cli_5xx":
		m.mxReqCli.Lock()
		m.ReqCountClient5xx += 1
		m.mxReqCli.Unlock()
	}

	return
}

func (m *MetricsHandler) PromReqInc(code, stype, path string) {
	m.PromExporter.httpReqTotal.WithLabelValues(stype, path, code).Inc()
}

// StartPush is a routing to dump/push metrics to
// anywhere (ToDo). Only stdout is supported atm.
func (m *MetricsHandler) StartPusher() {
	for {
		m.Time = time.Now()
		data, err := json.Marshal(m)
		if err != nil {
			log.Println("Error building metrics...")
			time.Sleep(5 * time.Second)
			continue
		}
		m.event.Send("metrics", "metrics-push", string(data))
		time.Sleep(1 * time.Second)
	}
}
