package metric

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/mtulio/go-lab-api/internal/event"
)

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
	mxReqService    sync.Mutex
	ReqCountService uint64 `json:"reqc_service"`
	mxReqHC         sync.Mutex
	ReqCountHC      uint64 `json:"reqc_hc"`
	mxReqCli        sync.Mutex
	ReqCountClient  uint64 `json:"reqc_client"`

	event *event.EventHandler
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
	}
	return
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
