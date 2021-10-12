package server

import (
	"sync"
	"time"
)

type MetricsHandler struct {
	Time                time.Time `json:"time"`
	AppHealthy          bool      `json:"app_healthy"`
	TargetHealthy       bool      `json:"tg_healthy"`
	TargetHealthCount   uint8     `json:"tg_health_count"`
	TargetUnhealthCount uint8     `json:"tg_unhealth_count"`
	ReqCountService     uint8     `json:"reqc_service"`
	ReqCountHC          uint8     `json:"reqc_hc"`
	mxReqService        sync.Mutex
	mxReqHC             sync.Mutex
}

func NewMetricHandler() *MetricsHandler {
	return &MetricsHandler{}
}

func (m *MetricsHandler) Set(metric string, value uint8) {
	return
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
	}
	return
}
