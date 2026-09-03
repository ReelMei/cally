package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cally/internal/api"
	"cally/internal/config"
	"cally/internal/room"
	"cally/internal/service"
)

type Server struct {
	cfg         *config.Config
	manager     *room.Manager
	roomService *service.RoomService
	httpServer  *http.Server
}

func NewServer(cfg *config.Config, manager *room.Manager, roomService *service.RoomService) *Server {
	router := api.NewRouter(cfg, manager, roomService)

	httpServer := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeoutSec) * time.Second,
		IdleTimeout:  time.Duration(cfg.IdleTimeoutSec) * time.Second,
	}

	return &Server{
		cfg:         cfg,
		manager:     manager,
		roomService: roomService,
		httpServer:  httpServer,
	}
}

func (s *Server) Start() error {
	shutdownComplete := make(chan struct{})

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		sig := <-sigChan

		slog.Info("shutdown signal received", "signal", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(ctx); err != nil {
			slog.Error("graceful server shutdown error", "error", err)
		} else {
			slog.Info("http server shut down gracefully")
		}

		close(shutdownComplete)
	}()

	slog.Info("server starting", "addr", s.cfg.Addr(), "env", s.cfg.Env)
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server ListenAndServe error: %w", err)
	}

	<-shutdownComplete
	slog.Info("server stopped completely")
	return nil
}
