package dlq

import (
	"time"
)

const (
	_defaultBufferSize     = 1000
	_defaultInitialBackoff = 1 * time.Second
	_defaultMaxBackoff     = 30 * time.Second
)

type config struct {
	BufferSize     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

func defaultConfig() *config {
	return &config{
		BufferSize:     _defaultBufferSize,
		InitialBackoff: _defaultInitialBackoff,
		MaxBackoff:     _defaultMaxBackoff,
	}
}

type Option func(*config)

func WithBufferSize(size int) Option {
	return func(c *config) { c.BufferSize = size }
}

func WithInitialBackoff(d time.Duration) Option {
	return func(c *config) { c.InitialBackoff = d }
}

func WithMaxBackoff(d time.Duration) Option {
	return func(c *config) { c.MaxBackoff = d }
}
