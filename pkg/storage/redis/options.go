package redis

import (
	"time"
)

const (
	_defaultPoolSize     = 20
	_defaultMinIdleConns = 10
	_defaultPoolTimeout  = 3 * time.Second
	_defaultCacheTTL     = time.Hour
)

type Config struct {
	Host         string
	Port         string
	Password     string
	DB           int
	TTL          time.Duration
	PoolSize     int
	MinIdleConns int
	PoolTimeout  time.Duration
}

type Option func(*Config)

func defaultConfigs(baseCfg Config) *Config {
	cfg := &Config{
		Host:         baseCfg.Host,
		Port:         baseCfg.Port,
		Password:     baseCfg.Password,
		DB:           baseCfg.DB,
		TTL:          baseCfg.TTL,
		PoolSize:     baseCfg.PoolSize,
		MinIdleConns: baseCfg.MinIdleConns,
		PoolTimeout:  baseCfg.PoolTimeout,
	}

	if cfg.TTL == 0 {
		cfg.TTL = _defaultCacheTTL
	}
	if cfg.PoolSize == 0 {
		cfg.PoolSize = _defaultPoolSize
	}
	if cfg.MinIdleConns == 0 {
		cfg.MinIdleConns = _defaultMinIdleConns
	}
	if cfg.PoolTimeout == 0 {
		cfg.PoolTimeout = _defaultPoolTimeout
	}

	return cfg
}

func WithTTL(ttl time.Duration) Option {
	return func(c *Config) {
		c.TTL = ttl
	}
}

func WithPoolSize(size int) Option {
	return func(c *Config) {
		c.PoolSize = size
	}
}

func WithMinIdleConns(conns int) Option {
	return func(c *Config) {
		c.MinIdleConns = conns
	}
}

func WithPoolTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.PoolTimeout = timeout
	}
}
