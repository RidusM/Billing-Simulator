package metric

import (
	"github.com/prometheus/client_golang/prometheus"
)

var _ Cache = (*cacheMetrics)(nil)

type cacheMetrics struct {
	hits      *prometheus.CounterVec
	misses    *prometheus.CounterVec
	evictions *prometheus.CounterVec
	errs      *prometheus.CounterVec
}

func newCacheMetrics(registry *promRegistry) *cacheMetrics {
	hits := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"type"},
	)

	misses := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"type"},
	)

	evictions := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_evictions_total",
			Help: "Total number of cache evictions",
		},
		[]string{"type", "reason"},
	)

	errs := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_errors_total",
			Help: "Total number of cache errors",
		},
		[]string{"type", "reason"},
	)

	registry.registry.MustRegister(hits, misses, evictions, errs)

	return &cacheMetrics{
		hits:      hits,
		misses:    misses,
		evictions: evictions,
		errs:      errs,
	}
}

func (m *cacheMetrics) Hit(cacheType string) {
	m.hits.WithLabelValues(cacheType).Add(1)
}

func (m *cacheMetrics) Miss(cacheType string) {
	m.misses.WithLabelValues(cacheType).Add(1)
}

func (m *cacheMetrics) Eviction(cacheType string, reason string) {
	m.evictions.WithLabelValues(cacheType, reason).Add(1)
}

func (m *cacheMetrics) Error(cacheType string, op string) {
	m.errs.WithLabelValues(cacheType, op).Add(1)
}
