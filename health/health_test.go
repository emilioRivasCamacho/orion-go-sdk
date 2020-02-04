package health

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

/*
func TestHealthchecks(t *testing.T) {

	svc := orion.New("healthTest")

	orion.RegisterHealthCheck(&Dependency{
		Name:    "good",
		Timeout: 10 * time.Second,
		CheckIsWorking: func() (string, *oerror.Error) {
			return "", nil
		},
	})

	orion.RegisterHealthCheck(&Dependency{
		Name:    "bad",
		Timeout: 10 * time.Second,
		CheckIsWorking: func() (string, *oerror.Error) {
			return "The check went bad", oerror.New(string(HC_CRIT))
		},
	})

	orion.RegisterHealthCheck(&Dependency{
		Name:    "warn",
		Timeout: 10 * time.Second,
		CheckIsWorking: func() (string, *oerror.Error) {
			return "The check went bad", oerror.New(string(HC_WARN))
		},
		IsInternal: true,
	})

	orion.RegisterHealthCheck(&Dependency{
		Name:    "timeout",
		Timeout: 1 * time.Second,
		CheckIsWorking: func() (string, *oerror.Error) {
			time.Sleep(2 * time.Second)
			return "", nil
		},
	})

	done := make(chan DependencyCheckResult, 10)

	go orion.Listen(func() {

		req := &orion.Request{}
		req.SetPath("/" + un + "/status.am-i-up")

		genres := &orion.Response{}
		orion.Call(req, genres)

		req = &orion.Request{}
		req.SetPath("/" + un + "/status.good")

		res := &DependencyResponse{}
		orion.Call(req, res)

		done <- res.Payload

		req = &orion.Request{}
		req.SetPath("/" + un + "/status.bad")

		orion.Call(req, res)

		done <- res.Payload

		req = &orion.Request{}
		req.Timeout = new(int)
		*req.Timeout = 3000
		req.SetPath("/" + un + "/status.timeout")

		orion.Call(req, res)

		done <- res.Payload

		req = &orion.Request{}
		req.SetPath("/" + un + "/status.warn")

		orion.Call(req, res)

		done <- res.Payload

		req2 := &AggregateRequest{}
		req2.Params.Type = new(AggregationType)
		*req2.Params.Type = INTERNAL

		req2.SetPath("/" + un + "/status.aggregate")

		res2 := &AggregateResponse{}
		orion.Call(req2, res2)

		assert.Equal(t, 1, len(res2.Payload))

		done <- res2.Payload[0]

		orion.Close()
	})

	result := <-done
	assert.Equal(t, HC_OK, result.Result)
	result = <-done
	assert.Equal(t, HC_CRIT, result.Result)
	result = <-done
	assert.Equal(t, HC_CRIT, result.Result)
	result = <-done
	assert.Equal(t, HC_WARN, result.Result)
	result = <-done
	assert.Equal(t, HC_WARN, result.Result)
}
*/

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
