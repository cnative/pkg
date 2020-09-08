package health

import (
	"fmt"
	"time"

	"github.com/cnative/pkg/log"
)

type (
	// Option configures choices
	Option interface {
		apply(*healthChecker)
	}
	optionFunc func(*healthChecker)
)

// BindPort configures the health service to listen on specified port
func BindPort(port uint) Option {
	return optionFunc(func(hc *healthChecker) {
		hc.bindAddress = fmt.Sprintf(":%d", port)
	})
}

// FailureThreshold configures failure threshold before reporting bad health for the service
func FailureThreshold(failureThreshold uint) Option {
	return optionFunc(func(hc *healthChecker) {
		hc.failureThreshold = failureThreshold
	})
}

// SuccessSleepInterval option configurs duration to sleep between successful probes
func SuccessSleepInterval(duration time.Duration) Option {
	return optionFunc(func(hc *healthChecker) {
		hc.successSleepInterval = duration
	})
}

// FailureSleepInterval option configurs duration to sleep between successful probes
func FailureSleepInterval(duration time.Duration) Option {
	return optionFunc(func(hc *healthChecker) {
		hc.failureSleepInterval = duration
	})
}

// Logger configures logger for health service
func Logger(l log.Logger) Option {
	return optionFunc(func(hc *healthChecker) {
		hc.logger = l.NamedLogger("health")
	})
}
