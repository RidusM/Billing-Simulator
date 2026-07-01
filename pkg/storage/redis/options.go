package redis

import "time"

const (
	_defaultPoolSize     = 20
	_defaultMinIdleConns = 10
	_defaultPoolTimeout  = 3 * time.Second
	_defaultCacheTTL     = time.Hour
)

type Config struct {
	Host     string
	Port     string
	Password string
	DB       int

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
		PoolSize:     _defaultPoolSize,
		MinIdleConns: _defaultMinIdleConns,
		PoolTimeout:  _defaultPoolTimeout,
	}

	if cfg.TTL == 0 {
		cfg.TTL = _defaultCacheTTL
	}

	return cfg
}

func WithPassword(password string) Option {
	return func(c *Config) {
		c.Password = password
	}
}

func WithDB(db int) Option {
	return func(c *Config) {
		c.DB = db
	}
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
