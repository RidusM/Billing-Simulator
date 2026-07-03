package transaction

import (
	"errors"
	"time"
)

const (
	_defaultMaxAttempts    = 3
	_defaultBaseRetryDelay = 100 * time.Millisecond
	_defaultMaxRetryDelay  = 5 * time.Second
)

type Config struct {
	MaxAttempts    int
	BaseRetryDelay time.Duration
	MaxRetryDelay  time.Duration
}

func defaultConfigs() *Config {
	return &Config{
		MaxAttempts:    _defaultMaxAttempts,
		BaseRetryDelay: _defaultBaseRetryDelay,
		MaxRetryDelay:  _defaultMaxRetryDelay,
	}
}

type Option func(*Config)

func WithMaxAttempts(attempts int) Option {
	return func(c *Config) {
		c.MaxAttempts = attempts
	}
}

func WithBaseRetryDelay(delay time.Duration) Option {
	return func(c *Config) {
		c.BaseRetryDelay = delay
	}
}

func WithMaxRetryDelay(delay time.Duration) Option {
	return func(c *Config) {
		c.MaxRetryDelay = delay
	}
}

func validateConfig(c *Config) error {
	if c.BaseRetryDelay > c.MaxRetryDelay {
		return errors.New("baseRetryDelay cannot exceed maxRetryDelay")
	}
	return nil
}
