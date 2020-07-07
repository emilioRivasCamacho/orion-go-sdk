package health

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gig/orion-go-sdk/interfaces"
	"github.com/gig/orion-go-sdk/transport/http2"
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

func InstallHealthcheck(t interfaces.Transport, endpointPath string) {
	if http2Transport, ok := t.(*http2.Transport); ok {
		http2Transport.GetRootRouter().Get(endpointPath, func(w http.ResponseWriter, r *http.Request) {
			summary := GetSummaryOfHealthChecks()

			if len(summary) == 0 {
				w.WriteHeader(200)
				// TODO: Handle this error
				_, _ = w.Write([]byte("OK"))
			} else {
				summaryString := "Error(s):\n"
				for _, err := range summary {
					summaryString = summaryString + err.Error() + "\n"
				}
				w.WriteHeader(500)
				// TODO: Handle this error
				_, _ = w.Write([]byte(summaryString))
			}
		})
	} else {
		log.Panic("we only support healthcheck for HTTP transports.")
	}
}
