package orion

import (
	"testing"

	"time"

	oerror "github.com/gig/orion-go-sdk/error"
	"github.com/gig/orion-go-sdk/health"
	"github.com/stretchr/testify/assert"
)

func TestHealthchecks(t *testing.T) {

	svc := New("healthTest")
	svc.RegisterToWatchdog = false
	svc.EnableStatusEndpoints = true
	un := UniqueName(svc.Name, svc.ID)

	svc.RegisterHealthCheck(&health.Dependency{
		Name:    "good",
		Timeout: 10 * time.Second,
		CheckIsWorking: func() (string, *oerror.Error) {
			return "", nil
		},
	})

	svc.RegisterHealthCheck(&health.Dependency{
		Name:    "bad",
		Timeout: 10 * time.Second,
		CheckIsWorking: func() (string, *oerror.Error) {
			return "The check went bad", oerror.New(string(health.HC_CRIT))
		},
	})

	svc.RegisterHealthCheck(&health.Dependency{
		Name:    "warn",
		Timeout: 10 * time.Second,
		CheckIsWorking: func() (string, *oerror.Error) {
			return "The check went bad", oerror.New(string(health.HC_WARN))
		},
		IsInternal: true,
	})

	svc.RegisterHealthCheck(&health.Dependency{
		Name:    "timeout",
		Timeout: 1 * time.Second,
		CheckIsWorking: func() (string, *oerror.Error) {
			time.Sleep(2 * time.Second)
			return "", nil
		},
	})

	done := make(chan health.DependencyCheckResult, 10)

	go svc.Listen(func() {

		req := &Request{}
		req.SetPath("/" + un + "/status.am-i-up")

		genres := &Response{}
		svc.Call(req, genres)

		req = &Request{}
		req.SetPath("/" + un + "/status.good")

		res := &health.DependencyResponse{}
		svc.Call(req, res)

		done <- res.Payload

		req = &Request{}
		req.SetPath("/" + un + "/status.bad")

		svc.Call(req, res)

		done <- res.Payload

		req = &Request{}
		req.Timeout = new(int)
		*req.Timeout = 3000
		req.SetPath("/" + un + "/status.timeout")

		svc.Call(req, res)

		done <- res.Payload

		req = &Request{}
		req.SetPath("/" + un + "/status.warn")

		svc.Call(req, res)

		done <- res.Payload

		req2 := &health.AggregateRequest{}
		req2.Params.Type = new(health.AggregationType)
		*req2.Params.Type = health.INTERNAL

		req2.SetPath("/" + un + "/status.aggregate")

		res2 := &health.AggregateResponse{}
		svc.Call(req2, res2)

		assert.Equal(t, 1, len(res2.Payload))

		done <- res2.Payload[0]

		svc.Close()
	})

	result := <-done
	assert.Equal(t, health.HC_OK, result.Result)
	result = <-done
	assert.Equal(t, health.HC_CRIT, result.Result)
	result = <-done
	assert.Equal(t, health.HC_CRIT, result.Result)
	result = <-done
	assert.Equal(t, health.HC_WARN, result.Result)
	result = <-done
	assert.Equal(t, health.HC_WARN, result.Result)
}
