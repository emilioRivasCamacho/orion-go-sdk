package checks

import (
	"errors"
	"time"

	"github.com/gig/orion-go-sdk/health"
	"github.com/gig/orion-go-sdk/interfaces"
)

// NatsHealthcheck prepares a healthcheck for nats transport.
func NatsHealthcheck(t interfaces.Transport) *health.Dependency {
	return &health.Dependency{
		Name:    "nats",
		Timeout: 100 * time.Millisecond,
		CheckIsWorking: func() (health.HealthCheckResult, error) {
			if t.IsOpen() {
				return health.HC_OK, nil
			}
			return health.HC_CRIT, errors.New("nats connection is not open")
		},
	}
}
