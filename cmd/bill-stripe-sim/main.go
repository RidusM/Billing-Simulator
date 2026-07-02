package main

import (
	"bill-stripe-sim/internal/app"
	"bill-stripe-sim/internal/config"
	"bill-stripe-sim/pkg/configurator"
	"bill-stripe-sim/pkg/logger"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var log logger.Logger
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			if log != nil {
				log.Error("PANIC RECOVERED",
					"panic", r,
					"stack", stack,
				)
			} else {
				fmt.Fprintf(os.Stderr, "PANIC RECOVERED:%v\n%s\n", r, stack)
			}
			os.Exit(1)
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var cfg config.Config
	if err := configurator.Load(&cfg); err != nil {
		return fmt.Errorf("config load: %w", err)
	}

	log, err := logger.NewZapAdapter(cfg.App.Name, cfg.Env,
		logger.WithRotation(cfg.Logger.Filename, cfg.Logger.MaxSize, cfg.Logger.MaxBackups, cfg.Logger.MaxAge),
	)
	if err != nil {
		return fmt.Errorf("logger init: %w", err)
	}

	log.LogAttrs(ctx, logger.InfoLevel, "starting application",
		logger.String("name", cfg.App.Name),
		logger.String("version", cfg.App.Version),
		logger.String("env", cfg.Env),
		logger.String("http_addr", net.JoinHostPort(cfg.HTTP.Host, cfg.HTTP.Port)),
	)

	if err = app.Run(ctx, &cfg, log); err != nil {
		if errors.Is(err, context.Canceled) {
			log.LogAttrs(ctx, logger.InfoLevel, "application stopped gracefully")
			return nil
		}
		return fmt.Errorf("app run: %w", err)
	}

	log.LogAttrs(ctx, logger.InfoLevel, "shutdown complete")
	return nil
}
