package server;

import (
	"os"
	"os/signal"
	"syscall"
	"time"
)

type HealthCheckController struct {
	// Healthy flag. Return true when the termination
	// flag is not set.
	Healthy bool

	SignalsReceived uint8

	// Return true when termination is in Progress
	TerminationInProgress bool

	// Timeout in seconds that Termination flag should be set
	TerminationTimeout float64

	// TerminationTimer is the counter when Termination
	// flag is set. It should not be 0.
	//TerminationTicker uint64

	TerminationStartTime time.Time

	events *chan string

	termChan *chan os.Signal
}

func NewHealthCheckController(events *chan string) *HealthCheckController {
	return &HealthCheckController{
		Healthy: true,
		TerminationInProgress: false,
		SignalsReceived: 0,
		TerminationTimeout: 300,
		TerminationStartTime: time.Now(),
		termChan: SetupSignal(),
		events: events,
	}
}


func (hc *HealthCheckController) RunSignalHandler() {

	<-*hc.termChan
	
	hc.SignalsReceived += 1
	if hc.SignalsReceived == 2 {
		// Do exit
		*hc.events<-("Second signal received, exiting")
		time.Sleep(1)
		
	}
	hc.Healthy = false
	hc.TerminationInProgress = !(hc.Healthy)

	// reset signal handler
	hc.termChan = SetupSignal()

	hc.TerminationStartTime = time.Now()
}

func (hc *HealthCheckController) RunTicker() {

	for {

		if !(hc.TerminationInProgress) {
			time.Sleep(10)
			continue
		}

		// Ttermination timeout
		if time.Since(hc.TerminationStartTime).Seconds() >= hc.TerminationTimeout  {
			hc.Healthy = true
			hc.TerminationInProgress = !(hc.Healthy)
			hc.SignalsReceived = 0
			*hc.events<-"Termination timeout...reseting healthy"
		}

		time.Sleep(1)
	}

}

func SetupSignal() (*chan os.Signal) {
	termChan := make(chan os.Signal)
	signal.Notify(termChan, syscall.SIGTERM)
	return &termChan
}