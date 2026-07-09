package metric

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var _ Kafka = (*kafkaMetrics)(nil)

type kafkaMetrics struct {
	messagesProcessed *prometheus.CounterVec
	messagesFailed    *prometheus.CounterVec
	duration  *prometheus.HistogramVec
}

func newKafkaMetrics(registry *promRegistry) *kafkaMetrics {
	processed := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_processed_total",
			Help: "Total number of processed Kafka messages",
		},
		[]string{"topic", "partition"},
	)

	failed := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_failed_total",
			Help: "Total number of failed Kafka messages",
		},
		[]string{"topic", "partition", "reason"},
	)

	duration := prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "kafka_message_processing_duration_seconds",
        Help:    "Time spent processing a single Kafka message",
        Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
    },
    []string{"topic", "partition"},
)

	registry.registry.MustRegister(processed, failed, duration)

	return &kafkaMetrics{
		messagesProcessed: processed,
		messagesFailed:    failed,
		duration:  duration,
	}
}

func (m *kafkaMetrics) ObserveDuration(topic string, partition int, duration time.Duration) {
	m.duration.WithLabelValues(topic, partitionString(partition)).Observe(duration.Seconds())
}

func (m *kafkaMetrics) MessageProcessed(topic string, partition int) {
	m.messagesProcessed.WithLabelValues(topic, partitionString(partition)).Add(1)
}

func (m *kafkaMetrics) MessageFailed(topic string, partition int, reason string) {
	m.messagesFailed.WithLabelValues(topic, partitionString(partition), reason).Add(1)
}

func partitionString(partition int) string {
	if partition == -1 {
		return "all"
	}
	return strconv.Itoa(partition)
}
