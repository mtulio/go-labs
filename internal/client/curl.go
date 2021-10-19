package client

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/mtulio/go-lab-api/internal/event"
	"github.com/mtulio/go-lab-api/internal/metric"
)

type Curl struct {
	cfg *CurlOptions
	m   *metric.MetricsHandler
	e   *event.EventHandler
}

type CurlOptions struct {
	Endpoint     string
	IntervalMs   uint64
	TimeoutSec   uint8
	SlowStartSec uint8
	Count        uint64
}

func NewCurlWithConfig(
	cfg *CurlOptions,
	m *metric.MetricsHandler,
	e *event.EventHandler) (*Curl, error) {

	c := Curl{
		cfg: cfg,
		m:   m,
		e:   e,
	}

	return &c, nil
}

func (c *Curl) Go() {
	return
}

// Loop will call URL (Go) according intervalMs
func (c *Curl) Loop(callback bool, callbackFN func(*http.Response)) {
	cfg := c.cfg
	appName := "curl"
	var reqCount uint64 = 0

	if c.cfg.SlowStartSec > 0 {
		msg := fmt.Sprintf("Starting client in  %ds", c.cfg.SlowStartSec)
		c.e.Send("request-client", appName, msg)
		time.Sleep(time.Duration(c.cfg.SlowStartSec) * time.Second)
	}

	for {
		reqCount += 1

		tlsCfg := tls.Config{InsecureSkipVerify: true}
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tlsCfg

		resp, err := http.Get(cfg.Endpoint)
		if err != nil {
			fixedDelay := 1 // backoff
			msg := fmt.Sprintf("ERROR received from server. Delaying %ds: %s", fixedDelay, err)
			c.e.Send("request-client", appName, msg)
			time.Sleep(time.Duration(fixedDelay) * time.Second)
			continue
		}

		c.m.Inc("requests_client")
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			c.m.Inc("requests_cli_2xx")
		} else if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			c.m.Inc("requests_cli_4xx")
		} else {
			c.m.Inc("requests_cli_5xx")
		}
		if callback {
			callbackFN(resp)
		}

		time.Sleep(time.Duration(cfg.IntervalMs) * time.Millisecond)
	}
}
