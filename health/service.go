package health

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/cnative/pkg/log"
)

type (
	// Probe is determines service health
	Probe interface {
		// Healthy checks if service is healthy or not
		Healthy() error
		Ready() (bool, error)
	}

	// Service for health check
	Service interface {
		// RegisterProbe a probe
		RegisterProbe(name string, p Probe)

		// Start health service
		Start() error

		// Stop health service
		Stop(ctx context.Context) error
	}

	healthChecker struct {
		server               *http.Server
		logger               log.Logger
		probes               map[string]Probe
		quit                 chan bool
		bindAddress          string
		failureThreshold     uint
		successSleepInterval time.Duration
		failureSleepInterval time.Duration
		mu                   sync.Mutex
		failureCount         uint
	}
)

func (f optionFunc) apply(hc *healthChecker) {
	f(hc)
}

// New creates a HealthService
func New(otions ...Option) Service {
	hc := &healthChecker{
		probes:               make(map[string]Probe),
		quit:                 make(chan bool),
		failureThreshold:     5,
		successSleepInterval: time.Second * 5,
		failureSleepInterval: time.Second * 2,
	}

	for _, opt := range otions {
		opt.apply(hc)
	}

	if hc.logger == nil {
		hc.logger = log.NewNop()
	}

	return hc
}

// Start HealthService
func (h *healthChecker) Start() error {
	go h.healthcheck()

	m := http.NewServeMux()

	m.HandleFunc("/live", h.livenessProbe)
	m.HandleFunc("/ready", h.readinessProbe)

	h.server = &http.Server{
		Addr:    h.bindAddress,
		Handler: m,
	}
	return h.server.ListenAndServe()
}

// Stop gracefully shuts down health service
func (h *healthChecker) Stop(ctx context.Context) error {
	if h.server != nil {
		return nil
	}
	h.quit <- true
	return h.server.Shutdown(ctx)
}

// healthcheck keeps checking the probes
func (h *healthChecker) healthcheck() {
	for {
		select {
		case <-h.quit:
			h.logger.Info("Stopping Health Service")
			break
		default:
			healthy := true
			h.mu.Lock()
			for name, probe := range h.probes {
				err := probe.Healthy()
				if err != nil {
					healthy = false
					h.logger.Warnf("Healthcheck failed for probe %s: %+v", name, err)
				}
			}
			h.mu.Unlock()

			sleepDuration := h.successSleepInterval
			if healthy {
				h.failureCount = 0
			} else {
				h.failureCount++
				sleepDuration = h.failureSleepInterval
			}

			time.Sleep(sleepDuration)
		}
	}
}

// livenessProbe to signal service termination.
func (h *healthChecker) livenessProbe(res http.ResponseWriter, req *http.Request) {
	if h.failureCount > h.failureThreshold {
		http.Error(res, "service unhealthy", http.StatusInternalServerError)
		return
	}
}

// readynessProbe is signal to indicate temporary unavailability so no live traffic is sent
func (h *healthChecker) readinessProbe(res http.ResponseWriter, req *http.Request) {
	if h.failureCount > 0 {
		http.Error(res, "service unhealthy", http.StatusInternalServerError)
		return
	}
}

// RegisterProbe adds a probe
func (h *healthChecker) RegisterProbe(name string, p Probe) {
	h.mu.Lock()
	h.probes[name] = p
	h.mu.Unlock()
}
