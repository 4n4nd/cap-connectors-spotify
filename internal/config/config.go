package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config represents runtime configuration supplied via environment variables.
type Config struct {
	HTTPPort            int
	LogLevel            string
	SpotifyClientID     string
	SpotifyClientSecret string
	SpotifyRedirectURI  string
	TokenEncKey         string
}

const (
	defaultHTTPPort = 8081
	defaultLogLevel = "info"
)

var supportedLogLevels = map[string]struct{}{
	"debug": {},
	"info":  {},
	"warn":  {},
	"error": {},
}

// Load gathers, normalises, and validates configuration.
func Load() (Config, error) {
	cfg := Config{
		HTTPPort: defaultHTTPPort,
		LogLevel: defaultLogLevel,
	}

	if portStr := os.Getenv("HTTP_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HTTP_PORT: %w", err)
		}
		cfg.HTTPPort = port
	}

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.LogLevel = strings.ToLower(level)
	}

	cfg.SpotifyClientID = os.Getenv("SPOTIFY_CLIENT_ID")
	cfg.SpotifyClientSecret = os.Getenv("SPOTIFY_CLIENT_SECRET")
	cfg.SpotifyRedirectURI = os.Getenv("SPOTIFY_REDIRECT_URI")
	cfg.TokenEncKey = os.Getenv("TOKEN_ENC_KEY")

	if cfg.HTTPPort <= 0 || cfg.HTTPPort > 65535 {
		return Config{}, fmt.Errorf("invalid HTTP port: %d", cfg.HTTPPort)
	}

	if _, ok := supportedLogLevels[cfg.LogLevel]; !ok {
		return Config{}, fmt.Errorf("invalid log level: %s", cfg.LogLevel)
	}

	return cfg, nil
}
