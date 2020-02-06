package health

import (
	"time"
)

const (
	// How often the loop will check that all dependencies are OK
	defaultHealthCheckLoopPeriod = 30 * time.Second
)

// LoopOverHealthChecks loops FOREVER over the HealthChecks that are given.
func LoopOverHealthChecks(dependencies []Dependency, close chan struct{}) {
	run := true
	delay := make(chan struct{}, 1)
	delay <- struct{}{}

	// This go function produces an event to the channel for the loop.
	go func() {
		for run {
			time.Sleep(defaultHealthCheckLoopPeriod)
			delay <- struct{}{}
		}
	}()

	for run {
		select {
		case <-close: // We want to end up the loop
			run = false
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
