package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"github.com/4n4nd/cap-connectors-spotify/internal/config"
	httpserver "github.com/4n4nd/cap-connectors-spotify/internal/http"
	"github.com/4n4nd/cap-connectors-spotify/internal/spotify"
	"github.com/4n4nd/cap-connectors-spotify/internal/store"
	"github.com/4n4nd/cap-connectors-spotify/internal/svc"
)

var version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	logger := buildLogger(cfg.LogLevel)
	logger.Info().
		Str("version", version).
		Int("port", cfg.HTTPPort).
		Msg("starting connectors-spotify")

	tokenStore := store.NewMemoryTokenStore()
	spotifyClient := spotify.NewClient(tokenStore)
	service := svc.New(spotifyClient)

	router := httpserver.NewRouter(logger, service)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info().Msg("HTTP server listening")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("server error")
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	logger.Info().Msg("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("graceful shutdown failed")
	} else {
		logger.Info().Msg("shutdown complete")
	}
}

func buildLogger(level string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	lvl := zerolog.InfoLevel
	switch level {
	case "debug":
		lvl = zerolog.DebugLevel
	case "warn":
		lvl = zerolog.WarnLevel
	case "error":
		lvl = zerolog.ErrorLevel
	}

	logger := zerolog.New(os.Stdout).Level(lvl).With().Timestamp().Logger()
	if lvl == zerolog.DebugLevel {
		logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	}
	return logger
}
