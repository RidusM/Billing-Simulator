package kafka

import (
	"errors"
	"time"
)

const (
	_defaultMaxAttempts    = 3
	_defaultBaseRetryDelay = 1 * time.Second
	_defaultMaxRetryDelay  = 30 * time.Second
	_defaultWorkersCount   = 1
)

type Config struct {
	MaxAttempts    int
	BaseRetryDelay time.Duration
	MaxRetryDelay  time.Duration
	WorkersCount   int
}

func DefaultConfig() Config {
	return Config{
		MaxAttempts:    _defaultMaxAttempts,
		BaseRetryDelay: _defaultBaseRetryDelay,
		MaxRetryDelay:  _defaultMaxRetryDelay,
		WorkersCount:   _defaultWorkersCount,
	}
}

func (c Config) Validate() error {
	if c.BaseRetryDelay > c.MaxRetryDelay {
		return errors.New("base_retry_delay cannot exceed max_retry_delay")
	}
	return nil
}

type ProcessorOption func(*Config)

func WithMaxAttempts(attempts int) ProcessorOption {
	return func(c *Config) {
		c.MaxAttempts = attempts
	}
}

func WithBaseRetryDelay(delay time.Duration) ProcessorOption {
	return func(c *Config) {
		c.BaseRetryDelay = delay
	}
}

func WithMaxRetryDelay(delay time.Duration) ProcessorOption {
	return func(c *Config) {
		c.MaxRetryDelay = delay
	}
}

func WithWorkersCount(count int) ProcessorOption {
	return func(c *Config) {
		c.WorkersCount = count
	}
}
