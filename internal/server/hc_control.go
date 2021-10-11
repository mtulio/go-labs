package server

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	//"log"
	"sync"
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

	Event *EventHandler
}

func NewHealthCheckController(ev *EventHandler) *HealthCheckController {
	hc := HealthCheckController{
		Healthy:               true,
		terminationInProgress: false,
		terminationTimeout:    30.0,
		Event:                 ev,
	}
	//signal.Notify(hc.termChan, syscall.SIGTERM)
	return &hc
}

func (hc *HealthCheckController) Start() {
	go hc.runSignalHandler()
	go hc.runTicker()
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
	hc.locker.Unlock()
}

func (hc *HealthCheckController) StartUnhealth() {
	hc.locker.Lock()

	// Set Start time only when Unhealthy is started
	if hc.Healthy {
		hc.UnhealthSince = time.Now()
	}
	hc.Healthy = false
	hc.locker.Unlock()
}

func (hc *HealthCheckController) StartTermination() {
	hc.locker.Lock()
	hc.terminationInProgress = true
	hc.terminationStartTime = time.Now()
	hc.locker.Unlock()
}

func (hc *HealthCheckController) StopTermination() {
	hc.locker.Lock()
	hc.terminationInProgress = false
	hc.locker.Unlock()
}

// Run Termination checker until timeout, then reset to
// Healthy state
func (hc *HealthCheckController) runTicker() {

	for {
		if !(hc.terminationInProgress) {
			time.Sleep(5 * time.Second)
			continue
		}

		// Timeout 300s
		if time.Since(hc.terminationStartTime).Seconds() >= 30 {
			//log.Println("Restoring to Healthy state...")
			hc.Event.Send("runtime", "hc-controller", "Restoring to Health State")
			hc.StartHealth()
			hc.StopTermination()
		}

		time.Sleep(1 * time.Second)
	}

}
