package health

import (
	"time"

	"os"

	oerror "github.com/gig/orion-go-sdk/error"
	"github.com/gig/orion-go-sdk/interfaces"
	"github.com/gig/orion-go-sdk/request"
	"github.com/gig/orion-go-sdk/response"
)

const (
	// How often the loop will check that all dependencies are OK
	defaultHealthCheckLoopPeriod = 30 * time.Second
)

// LoopOverHealthChecks loops FOREVER over the HealthChecks that are given. 
func LoopOverHealthChecks(dependencies []Dependency) {
	for {
		ResetSummaryOfHealthChecks()
		// TODO: check dependencies
		// TODO: AppendHealthCheckError(error)
		time.Sleep(defaultHealthCheckLoopPeriod)
	}
}