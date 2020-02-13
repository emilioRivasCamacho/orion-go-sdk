package health

import (
	"errors"
	"sync"
	"time"
)

type HealthCheckResult string

const (
	HC_OK   HealthCheckResult = "OK"
	HC_WARN HealthCheckResult = "WARN"
	HC_CRIT HealthCheckResult = "CRIT"
)

type Dependency struct {
	CheckIsWorking func() (HealthCheckResult, error)
	Timeout        time.Duration
	Name           string
}

var (
	// The latest summary of errors happening when running health checks.
	summary      []error
	summaryMutex sync.RWMutex
)

func GetSummaryOfHealthChecks() []error {
	summaryMutex.RLock()
	defer summaryMutex.RUnlock()
	return summary
}

func ResetSummaryOfHealthChecks() {
	summaryMutex.Lock()
	defer summaryMutex.Unlock()
	summary = []error{}
}

func AppendHealthCheckError(err error) {
	summaryMutex.Lock()
	defer summaryMutex.Unlock()
	summary = append(summary, err)
}

func WrapHealthCheckWithATimeout(name string, timeout time.Duration, check func() (HealthCheckResult, error)) func() (HealthCheckResult, error) {
	return func() (HealthCheckResult, error) {
		resultChannel := make(chan struct {
			str HealthCheckResult
			err error
		})
		timeoutChannel := make(chan bool)

		go func() {
			str, err := check()
			resultChannel <- struct {
				str HealthCheckResult
				err error
			}{
				str: str,
				err: err,
			}
		}()

		go func() {
			time.Sleep(timeout)
			timeoutChannel <- true
		}()

		select {
		case res := <-resultChannel:
			return res.str, res.err
		case <-timeoutChannel:
			err := errors.New("the health check " + name + " did timeout for " + string(timeout/time.Second) + " seconds")
			return HC_CRIT, err
		}
	}
}
