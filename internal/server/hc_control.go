package server

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"sync"

	"github.com/mtulio/go-lab-api/internal/event"
	"github.com/mtulio/go-lab-api/internal/metric"
)

type HealthCheckController struct {
	// Healthy flag. Return true when the termination
	// flag is not set.
	Healthy bool

	HealthSince   time.Time
	UnhealthSince time.Time

	// Return true when termination is in Progress
	terminationInProgress bool

	// Timeout in seconds that Termination flag should be set
	terminationTimeout float64

	// TerminationTimer is the counter when Termination
	// flag is set. It should not be 0.
	terminationStartTime time.Time

	//events *chan string

	//termChan chan os.Signal

	// mutex
	locker sync.Mutex

	Event *event.EventHandler

	Metric *metric.MetricsHandler
}

type HCControllerOpts struct {
	Event       *event.EventHandler
	Metric      *metric.MetricsHandler
	TermTimeout uint64
}

func NewHealthCheckController(op *HCControllerOpts) *HealthCheckController {

	hc := HealthCheckController{
		Healthy:               true,
		terminationInProgress: false,
		//terminationTimeout:    (time.Duration(float64(op.TermTimeout)) * time.Second),
		terminationTimeout: float64(op.TermTimeout),
		Event:              op.Event,
		Metric:             op.Metric,
	}
	hc.Metric.AppHealthy = hc.Healthy
	hc.Metric.AppTermination = hc.terminationInProgress
	return &hc
}

func (hc *HealthCheckController) Start() {
	go hc.runSignalHandler()
	go hc.runTicker()
}

func (hc *HealthCheckController) GetHealthy() bool {
	return hc.Healthy
}

// Returns healthy/unhealthy string
func (hc *HealthCheckController) GetHealthyStr() string {
	if hc.Healthy {
		return "healthy"
	}
	return "unhealthy"
}

func (hc *HealthCheckController) StartHealth() {
	// when state is changed to healthy, all
	// termination operations will be clear
	hc.locker.Lock()
	hc.Healthy = true
	hc.HealthSince = time.Now()
	hc.Metric.AppHealthy = hc.Healthy
	hc.locker.Unlock()
}

func (hc *HealthCheckController) StartUnhealth() {
	hc.locker.Lock()

	// Set Start time only when Unhealthy is started
	if hc.Healthy {
		hc.UnhealthSince = time.Now()
	}
	hc.Healthy = false
	hc.Metric.AppHealthy = hc.Healthy
	hc.locker.Unlock()
}

func (hc *HealthCheckController) StartTermination() {
	hc.locker.Lock()
	hc.terminationInProgress = true
	hc.terminationStartTime = time.Now()
	hc.Metric.AppTermination = hc.terminationInProgress
	hc.locker.Unlock()
}

func (hc *HealthCheckController) StopTermination() {
	hc.locker.Lock()
	hc.terminationInProgress = false
	hc.Metric.AppTermination = hc.terminationInProgress
	hc.locker.Unlock()
}

// Handle SiGTERM signal, if it was sent twice the termination
// will be forced. Otherwise the timeout ticket will clear the
// process for a while.
func (hc *HealthCheckController) runSignalHandler() {

	for {
		msg := ("Running Signal handler")
		hc.Event.Send("runtime", "hc-controller", msg)

		termChan := make(chan os.Signal)
		signal.Notify(termChan, syscall.SIGTERM)

		<-termChan

		msg = ("Termination Signal receievd")
		hc.Event.Send("runtime", "hc-controller", msg)

		if hc.terminationInProgress {
			msg = ("Termination already in progress, forcing termination.")
			hc.Event.Send("runtime", "hc-controller", msg)
			os.Exit(0)
		}

		hc.StartTermination()
		hc.StartUnhealth()

		termChan = make(chan os.Signal)
		signal.Notify(termChan, syscall.SIGTERM)
	}
}

// Run Termination checker until timeout, then reset to
// Healthy state
func (hc *HealthCheckController) runTicker() {

	for {
		if !(hc.terminationInProgress) {
			time.Sleep(5 * time.Second)
			continue
		}

		// Timeout (arg --termination-timeout)
		if time.Since(hc.terminationStartTime).Seconds() >= hc.terminationTimeout {
			//log.Println("Restoring to Healthy state...")
			hc.Event.Send("runtime", "hc-controller", "Restoring to Health State")
			hc.StartHealth()
			hc.StopTermination()
		}

		time.Sleep(1 * time.Second)
	}

}
