package metric

import (
	"net/http"
	"time"
)

type (
	Factory interface {
		HTTP() HTTP
		Transaction() Transaction
		Cache() Cache
		Kafka() Kafka
		DLQ() DLQ
		Business() Business
		Handler() http.Handler
	}

	HTTP interface {
		Request(method, path string, status int, duration time.Duration)
	}

	Transaction interface {
		ObserveDuration(operation string, duration time.Duration)
		IncrementRetries(operation string)
		IncrementFailures(operation string)
		IncrementActive(operation string)
		DecrementActive(operation string)
	}

	Cache interface {
		Hit(cacheType string)
		Miss(cacheType string)
		Eviction(cacheType string, reason string)
		Error(cacheType string, op string)
	}

	Kafka interface {
		ObserveDuration(topic string, partition int, duration time.Duration)
		MessageProcessed(topic string, partition int)
		MessageFailed(topic string, partition int, reason string)
	}

	DLQ interface {
		DLSent(dlqTopic string, originalTopic string)
		DLError(dlqTopic string, reason string)
		DLRetryCount(originalTopic string, retryCount int)
	}

	Business interface {
		EventCreated()
		EventDeleted()
		EventArchived(count int)
		ReminderScheduled()
		ReminderFired()
	}
)
