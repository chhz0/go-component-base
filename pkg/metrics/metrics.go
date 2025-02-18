package metrics

import (
	"sort"
	"sync"
	"sync/atomic"
)

type MetricType int

const (
	Counter MetricType = iota
	Gauge
	Histogram
	Summary
)

type Metric interface {
	Name() string
	Type() MetricType
	Value() interface{}
}

// Collector is a collection of metrics
type Collector struct {
	mu      sync.RWMutex
	metrics map[string]Metric
}

var globalCollector = NewCollector()

func NewCollector() *Collector {
	return &Collector{
		metrics: make(map[string]Metric),
	}
}

func GlobalCollector() *Collector {
	return globalCollector
}

// CounterMetric(计数器) is a metric that represents a counter
type CounterMetric struct {
	name  string
	value atomic.Uint64
}

func (c *CounterMetric) Name() string       { return c.name }
func (c *CounterMetric) Type() MetricType   { return Counter }
func (c *CounterMetric) Value() interface{} { return c.value.Load() }
func (c *CounterMetric) Inc()               { c.value.Add(1) }
func (c *CounterMetric) Add(delta uint64)   { c.value.Add(delta) }
func (c *CounterMetric) Reset()             { c.value.Store(0) }

// GaugeMetric(仪表盘) is a metric that represents a single value
type GaugeMetric struct {
	name  string
	value atomic.Int64
}

func (g *GaugeMetric) Name() string       { return g.name }
func (g *GaugeMetric) Type() MetricType   { return Gauge }
func (g *GaugeMetric) Value() interface{} { return g.value.Load() }
func (g *GaugeMetric) Set(value int64)    { g.value.Store(value) }
func (g *GaugeMetric) Add(delta int64)    { g.value.Add(delta) }
func (g *GaugeMetric) Sub(delta int64)    { g.value.Add(-delta) }

// HistogramMetric(直方图) is a metric that represents a histogram
type HistogramMetric struct {
	name    string
	buckets []float64
	counts  []float64
	count   atomic.Uint64
	sum     float64
	mu      sync.Mutex
}

func (h *HistogramMetric) Name() string     { return h.name }
func (h *HistogramMetric) Type() MetricType { return Histogram }
func (h *HistogramMetric) Value() interface{} {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]float64(nil), h.counts...)
}
func (h *HistogramMetric) Observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.buckets) == 0 {
		return
	}

	h.count.Add(1)
	h.sum += value
	for i, bucket := range h.buckets {
		if value <= bucket {
			h.counts[i]++
			continue
		}
	}
	// if value > h.buckets[len(h.buckets)-1] {
	// 	h.counts[len(h.counts)-1]++
	// }
}
func (h *HistogramMetric) Count() uint64 {

	return h.count.Load()
}
func (h *HistogramMetric) Sum() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sum
}
func (h *HistogramMetric) Buckets() []float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]float64(nil), h.buckets...)
}

// SummaryMetric(摘要) is a metric that represents a summary
type SummaryMetric struct {
	name      string
	quantiles map[float64]float64
	values    []float64
	sum       float64
	count     atomic.Uint64
	mu        sync.Mutex
}

func (s *SummaryMetric) Name() string     { return s.name }
func (s *SummaryMetric) Type() MetricType { return Summary }
func (s *SummaryMetric) Value() interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]float64(nil), s.values...)
}
func (s *SummaryMetric) Observe(value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.values = append(s.values, value)
	s.sum += value
	s.count.Add(1)
}
func (s *SummaryMetric) Count() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count.Load()
}
func (s *SummaryMetric) Sum() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sum
}
func (s *SummaryMetric) Quantiles() map[float64]float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	quantiles := make(map[float64]float64)
	if len(s.values) == 0 {
		return quantiles
	}

	sortedValues := make([]float64, len(s.values))
	copy(sortedValues, s.values)
	sort.Float64s(sortedValues)

	for q := range s.quantiles {
		index := int(q * float64(len(sortedValues)))
		if index >= len(sortedValues) {
			index = len(sortedValues) - 1
		}
		quantiles[q] = sortedValues[index]
	}

	return quantiles
}

func (s *SummaryMetric) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values = s.values[:0]
	s.sum = 0
	s.count.Store(0)
}

func (c *Collector) Register(m Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.metrics[m.Name()]; !ok {
		c.metrics[m.Name()] = m
	}
}

func (c *Collector) Get(name string) Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.metrics[name]
}

func (c *Collector) Metrics() map[string]Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot := make(map[string]Metric, len(c.metrics))
	for k, m := range c.metrics {
		snapshot[k] = m
	}
	return snapshot
}

func NewCounter(name string) *CounterMetric {
	c := &CounterMetric{name: name}
	return c
}

func NewGauge(name string) *GaugeMetric {
	g := &GaugeMetric{name: name}
	return g
}

func NewHistogram(name string, buckets []float64) *HistogramMetric {
	sort.Float64s(buckets)
	h := &HistogramMetric{
		name:    name,
		buckets: buckets,
		sum:     0,
		counts:  make([]float64, len(buckets), len(buckets)*2),
	}
	return h
}

func NewSummary(name string, quantiles map[float64]float64) *SummaryMetric {
	s := &SummaryMetric{
		name:      name,
		quantiles: quantiles,
		values:    make([]float64, 0, 100),
		sum:       0,
		count:     atomic.Uint64{},
	}
	return s
}
