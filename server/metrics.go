package server

import (
	"time"

	"go.opentelemetry.io/contrib/instrumentation/host"
	rt "go.opentelemetry.io/contrib/instrumentation/runtime"
)

// ProcessMetricsCollector GO Metrics collector
type ProcessMetricsCollector interface {
	Start() error
	Stop()
}
type processMetricsCollector struct {
	period time.Duration // time interval between calls to runtime.ReadMemStats()
}

func (p *processMetricsCollector) Start() error {

	// host metrics
	if err := host.Start(); err != nil {
		return err
	}

	// host.WithMeterProvider()

	// go process metrics
	return rt.Start(rt.WithMinimumReadMemStatsInterval(p.period))
}

func (p *processMetricsCollector) Stop() {
}

// NewProcessMetricsCollector collects metrics at process level
func NewProcessMetricsCollector() ProcessMetricsCollector {
	return &processMetricsCollector{period: defaultProcessMetricsCollectionFrequency}
}
