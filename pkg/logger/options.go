package logger

import (
	"io"
	"os"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	_defaultMaxSize    = 100
	_defaultMaxBackups = 7
	_defaultMaxAge     = 30
)

type Config struct {
	Level      Level
	AppName    string
	Env        string
	Filename   string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
	Stdout     bool

	writer     io.Writer
	writerOnce sync.Once
}

type Option func(*Config)

func defaultConfigs() *Config {
	return &Config{
		Level:      InfoLevel,
		MaxSize:    _defaultMaxSize,
		MaxBackups: _defaultMaxBackups,
		MaxAge:     _defaultMaxAge,
		Compress:   true,
		Stdout:     true,
	}
}

func WithLevel(l Level) Option {
	return func(c *Config) { c.Level = l }
}

func WithRotation(filename string, maxSize, maxBackups, maxAge int) Option {
	return func(c *Config) {
		c.Filename = filename
		c.MaxSize = maxSize
		c.MaxBackups = maxBackups
		c.MaxAge = maxAge
	}
}

func (c *Config) GetWriter() io.Writer {
	c.writerOnce.Do(func() {
		var writers []io.Writer
		if c.Stdout {
			writers = append(writers, os.Stdout)
		}

		if c.Filename != "" {
			writers = append(writers, &lumberjack.Logger{
				Filename:   c.Filename,
				MaxSize:    c.MaxSize,
				MaxBackups: c.MaxBackups,
				MaxAge:     c.MaxAge,
				Compress:   c.Compress,
			})
		}

		if len(writers) == 0 {
			c.writer = io.Discard
		} else {
			c.writer = io.MultiWriter(writers...)
		}
	})

	return c.writer
}
