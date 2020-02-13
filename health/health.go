package health

import (
	"sync"
	"time"
)

const (
	// How often the loop will check that all dependencies are OK
	defaultHealthCheckLoopPeriod = 30 * time.Second
)

// LoopOverHealthChecks loops FOREVER over the HealthChecks that are given.
func LoopOverHealthChecks(dependencies []Dependency, close chan struct{}) {
	run := true
	runLock := sync.Mutex{}
	delay := make(chan struct{}, 1)
	delay <- struct{}{}

	// This go function produces an event to the channel for the loop.
	go func() {
		runLock.Lock()
		for run {
			runLock.Unlock()
			time.Sleep(defaultHealthCheckLoopPeriod)
			delay <- struct{}{}
			runLock.Lock()
		}
		runLock.Unlock()
	}()

	for run {
		select {
		case <-close: // We want to end up the loop
			runLock.Lock()
			run = false
			runLock.Unlock()
		case <-delay: // Every loop period we run the health checks.
			ResetSummaryOfHealthChecks()
			for _, dep := range dependencies {
				res, err := dep.CheckIsWorking()
				if res == HC_CRIT {
					AppendHealthCheckError(err)
				} else if res == HC_WARN {
					// TODO: Log there's a warning with a health check
				}
			}

		}
	}

}
