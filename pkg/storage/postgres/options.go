package postgres

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"
)

const (
	_defaultConnAttempts   = 10
	_defaultBaseRetryDelay = 100 * time.Millisecond
	_defaultMaxRetryDelay  = 5 * time.Second
	_defaultMaxPoolSize    = 100
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string

	MaxPoolSize    int32
	ConnAttempts   int
	BaseRetryDelay time.Duration
	MaxRetryDelay  time.Duration
}

func (c Config) DSN() string {
    return fmt.Sprintf(
        "postgres://%s:%s@%s/%s?sslmode=%s",
        url.PathEscape(c.User),
        url.PathEscape(c.Password),
        net.JoinHostPort(c.Host, c.Port),
        url.PathEscape(c.Name),
        c.SSLMode,
    )
}

func defaultConfigs(baseCfg Config) *Config {
	cfg := &Config{
		Host:           baseCfg.Host,
		Port:           baseCfg.Port,
		User:           baseCfg.User,
		Password:       baseCfg.Password,
		Name:           baseCfg.Name,
		SSLMode:        baseCfg.SSLMode,
		MaxPoolSize:    _defaultMaxPoolSize,
		ConnAttempts:   _defaultConnAttempts,
		BaseRetryDelay: _defaultBaseRetryDelay,
		MaxRetryDelay:  _defaultMaxRetryDelay,
	}
	return cfg
}

type Option func(*Config)

func WithMaxPoolSize(size int32) Option {
	return func(c *Config) {
		c.MaxPoolSize = size
	}
}

func WithConnAttempts(attempts int) Option {
	return func(c *Config) {
		c.ConnAttempts = attempts
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
