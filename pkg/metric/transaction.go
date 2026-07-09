package metric

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var _ Transaction = (*transactionMetrics)(nil)

type transactionMetrics struct {
	duration *prometheus.HistogramVec
	active *prometheus.GaugeVec
	retries  *prometheus.CounterVec
	failures *prometheus.CounterVec
}

func newTransactionMetrics(registry *promRegistry) *transactionMetrics {
	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_transaction_duration_seconds",
			Help:    "Duration of database transactions in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1.0, 2.0, 5.0},
		},
		[]string{"operation"},
	)

	active := prometheus.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "db_transaction_active",
        Help: "Number of currently active transactions",
    },
    []string{"operation"},
)

	retries := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_transaction_retries_total",
			Help: "Total number of transaction retries",
		},
		[]string{"operation"},
	)

	failures := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_transaction_failures_total",
			Help: "Total number of failed transactions",
		},
		[]string{"operation"},
	)

	registry.registry.MustRegister(duration, active, retries, failures)

	return &transactionMetrics{
		duration: duration,
		active: active,
		retries:  retries,
		failures: failures,
	}
}

func (m *transactionMetrics) ObserveDuration(operation string, duration time.Duration) {
	m.duration.WithLabelValues(operation).Observe(duration.Seconds())
}

func (m *transactionMetrics) IncrementActive(operation string) {
	m.active.WithLabelValues(operation).Add(1)
}

func (m *transactionMetrics) DecrementActive(operation string) {
	m.active.WithLabelValues(operation).Add(-1)
}

func (m *transactionMetrics) IncrementRetries(operation string) {
	m.retries.WithLabelValues(operation).Add(1)
}

func (m *transactionMetrics) IncrementFailures(operation string) {
	m.failures.WithLabelValues(operation).Add(1)
}
