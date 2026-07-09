package metric

import (
	"github.com/prometheus/client_golang/prometheus"
)

var _ Business = (*businessMetrics)(nil)

type businessMetrics struct {
	eventCreated      prometheus.Counter
	eventDeleted      prometheus.Counter
	eventArchived     prometheus.Counter
	reminderScheduled prometheus.Counter
	reminderFired     prometheus.Counter
}

func newBusinessMetrics(registry *promRegistry) *businessMetrics {
	created := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "calendar_events_created_total",
			Help: "Total number of created events",
		})

	deleted := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "calendar_events_deleted_total",
			Help: "Total number of deleted events",
		})

	archived := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "calendar_events_archived_total",
			Help: "Total number of archived events",
		})

	reminderScheduled := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "calendar_reminders_scheduled_total",
			Help: "Total number of scheduled reminders",
		})

	reminderFired := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "calendar_reminders_fired_total",
			Help: "Total number of fired reminders",
		})

	registry.registry.MustRegister(created, deleted, archived, reminderScheduled, reminderFired)

	return &businessMetrics{
		eventCreated:      created,
		eventDeleted:      deleted,
		eventArchived:     archived,
		reminderScheduled: reminderScheduled,
		reminderFired:     reminderFired,
	}
}

func (m *businessMetrics) EventCreated() {
	m.eventCreated.Inc()
}

func (m *businessMetrics) EventDeleted() {
	m.eventDeleted.Inc()
}

func (m *businessMetrics) EventArchived(count int) {
	if count > 0 {
		m.eventArchived.Add(float64(count))
	}
}

func (m *businessMetrics) ReminderScheduled() {
	m.reminderScheduled.Inc()
}

func (m *businessMetrics) ReminderFired() {
	m.reminderFired.Inc()
}
