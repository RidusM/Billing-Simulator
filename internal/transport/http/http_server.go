package http

import (
	"bill-stripe-sim/internal/config"
	"bill-stripe-sim/pkg/logger"
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

type Server struct {
	server          *http.Server
	shutdownTimeout time.Duration
	log             logger.Logger
}

func NewServer(
	handler BillingHandler,
	cfg *config.HTTP,
	log logger.Logger,
) *Server {
	return &Server{
		server: &http.Server{
			Addr:              net.JoinHostPort(cfg.Host, cfg.Port),
			Handler:           handler.Engine(),
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			MaxHeaderBytes:    cfg.MaxHeaderBytes,
		},
		shutdownTimeout: cfg.ShutdownTimeout,
		log:             log,
	}
}

func (s *Server) Start(ctx context.Context) error {
	const op = "transport.http.Server.Start"

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		s.log.LogAttrs(ctx, logger.InfoLevel, "starting HTTP server",
			logger.String("addr", s.server.Addr),
		)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("%s: %w", op, err)
		}
		return nil
	})

	eg.Go(func() error {
		<-ctx.Done()
		return s.Stop(context.Background())
	})

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	const op = "transport.http.Server.Stop"

	shutdownCtx, cancel := context.WithTimeout(ctx, s.shutdownTimeout)
	defer cancel()

	s.log.LogAttrs(ctx, logger.InfoLevel, "shutting down HTTP server")

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		s.log.LogAttrs(ctx, logger.ErrorLevel, "HTTP server forced shutdown",
			logger.Any("error", err),
		)
		return fmt.Errorf("%s: %w", op, err)
	}

	s.log.LogAttrs(ctx, logger.InfoLevel, "HTTP server stopped gracefully")
	return nil
}
