package server

import (
	"context"
	rt "runtime"
	"runtime/debug"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

var (
	processMetricCost   = stats.Float64("process/metric_cost", "cost of collecting this metrics", "ms")
	processGoRoutines   = stats.Int64("process/go_routines", "number of go routines. useful to determine leaks over time", "1")
	processMemAlloc     = stats.Int64("process/mem_alloc", "process memory allocation", "By")
	processMemHeapAlloc = stats.Int64("process/mem_heap_alloc", "process memory heap allocation", "By")
	processGC           = stats.Int64("process/gc", "number of gc pauses", "1")
	processGCPause      = stats.Float64("process/gc_pause", "gc pause distribution", "ms")
)

var (
	// ProcessMetricCostView cost of metric collection in nanoseconds
	processMetricCostView = &view.View{
		Name:        processMetricCost.Name(),
		Measure:     processMetricCost,
		Description: "The distribution of the latencies of metric collection",
		// Latency in buckets:
		// [>=0ms, >=5ms, >=10ms, >=15ms, >=20ms, >=25ms, >=50ms, >=100ms]
		Aggregation: view.Distribution(0, 5, 10, 15, 20, 25, 50, 100),
	}

	// metric to represent the number of GO routines
	processGoRoutinesView = &view.View{
		Name:        processGoRoutines.Name(),
		Measure:     processGoRoutines,
		Description: "The number go routines",
		Aggregation: view.LastValue(),
	}

	// metric to represent memory allocated in bytes
	processMemAllocView = &view.View{
		Name:        processMemAlloc.Name(),
		Measure:     processMemAlloc,
		Description: "The memory allocated in bytes",
		Aggregation: view.LastValue(),
	}

	// metric to represent memory heap allocated in bytes
	processMemHeapAllocView = &view.View{
		Name:        processMemHeapAlloc.Name(),
		Measure:     processMemHeapAlloc,
		Description: "The memory heap allocated in bytes",
		Aggregation: view.LastValue(),
	}

	// metric to represent number of GC pauses
	processGCView = &view.View{
		Name:        processGC.Name(),
		Measure:     processGC,
		Description: "The number of GC pauses",
		Aggregation: view.LastValue(),
	}

	// metric to represent number of GC pauses
	processGCPauseView = &view.View{
		Name:        processGCPause.Name(),
		Measure:     processGCPause,
		Description: "The distribution of GC pause latencies",
		// Latency in buckets:
		// [>=0ms, >=5ms, >=10ms, >=15ms, >=20ms, >=25ms, >=50ms, >=100ms, >=200ms, >=400ms, >=1s]
		Aggregation: view.Distribution(0, 5, 10, 15, 20, 25, 50, 100, 200, 400, 1000),
	}
)

// ProcessMetricsCollector GO Metrics collector
type ProcessMetricsCollector interface {
	Start() error
	Stop()
}
type processMetricsCollector struct {
	period time.Duration
	done   chan bool
	lastGC time.Time
}

// DefaultProcessViews are the default go process views provided by this package.
var DefaultProcessViews = []*view.View{
	processMetricCostView,
	processGoRoutinesView,
	processMemAllocView,
	processMemHeapAllocView,
	processGCView,
	processGCPauseView,
}

func (p *processMetricsCollector) record() {
	// get a variety of runtime stats
	start := time.Now()
	numGoRoutines := rt.NumGoroutine()
	var memStats rt.MemStats
	rt.ReadMemStats(&memStats)
	gcStats := &debug.GCStats{}
	debug.ReadGCStats(gcStats)
	duration := float64(time.Since(start).Nanoseconds()) / 1e6
	ctx := context.Background()

	stats.Record(ctx, processMetricCost.M(duration))
	stats.Record(ctx, processGoRoutines.M(int64(numGoRoutines)))
	stats.Record(ctx, processMemAlloc.M(int64(memStats.Alloc)))
	stats.Record(ctx, processMemHeapAlloc.M(int64(memStats.HeapAlloc)))
	stats.Record(ctx, processGC.M(int64(gcStats.NumGC)))

	if len(gcStats.Pause) > 0 && !gcStats.LastGC.Equal(p.lastGC) {
		lastPauseTime := float64(gcStats.Pause[0].Nanoseconds()) / 1e6
		stats.Record(ctx, processGCPause.M(lastPauseTime))
		p.lastGC = gcStats.LastGC
	}
}

func (p *processMetricsCollector) Start() error {

	ticker := time.NewTicker(p.period)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.record()
		case <-p.done:
			return nil
		}
	}
}

func (p *processMetricsCollector) Stop() {
	if p.done != nil {
		close(p.done)
	}
}

// NewProcessMetricsCollector collects metrics at process level
func NewProcessMetricsCollector() ProcessMetricsCollector {
	return &processMetricsCollector{period: defaultProcessMetricsCollectionFrequency}
}
