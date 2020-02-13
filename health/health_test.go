package health

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoopOverHealthChecks(t *testing.T) {
	t.Run("Closing empty list", func(t *testing.T) {
		closeLoop := make(chan struct{})
		waitForClose := make(chan struct{})

		go func() {
			LoopOverHealthChecks([]Dependency{}, closeLoop)
			waitForClose <- struct{}{}
		}()

		closeLoop <- struct{}{}
		<-waitForClose
	})

	t.Run("All healthchecks are called", func(t *testing.T) {
		depCalled := make(chan struct{})
		closeLoop := make(chan struct{})
		waitForClose := make(chan struct{})

		deps := []Dependency{
			Dependency{
				Name:    "good",
				Timeout: 10 * time.Second,
				CheckIsWorking: func() (HealthCheckResult, error) {
					depCalled <- struct{}{}
					return HC_OK, nil
				},
			},

			Dependency{
				Name:    "bad",
				Timeout: 10 * time.Second,
				CheckIsWorking: func() (HealthCheckResult, error) {
					depCalled <- struct{}{}
					return HC_CRIT, errors.New("the check went bad")
				},
			},

			Dependency{
				Name:    "warn",
				Timeout: 10 * time.Second,
				CheckIsWorking: func() (HealthCheckResult, error) {
					depCalled <- struct{}{}
					return HC_WARN, errors.New("the check went bad")
				},
			},

			Dependency{
				Name:    "timeout",
				Timeout: 1 * time.Second,
				CheckIsWorking: WrapHealthCheckWithATimeout("timeout", 1*time.Second, func() (HealthCheckResult, error) {
					depCalled <- struct{}{}
					time.Sleep(2 * time.Second)
					return "", nil
				}),
			}}

		go func() {
			LoopOverHealthChecks(deps, closeLoop)
			waitForClose <- struct{}{}
		}()

		// Wait for every healthCheck to be called
		<-depCalled
		<-depCalled
		<-depCalled
		<-depCalled

		// Then kill the loop
		closeLoop <- struct{}{}
		<-waitForClose

		// Then check that there are two errors:

		summary := GetSummaryOfHealthChecks()
		fmt.Println(summary[1])
		assert.Len(t, summary, 2)
	})
}
