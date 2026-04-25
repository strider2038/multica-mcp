package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	MulticaBaseURL string
	MulticaToken   string
	MCPTransport   string
	LogLevel       string
	HTTPPort       int
	ReadOnly       bool
	MCPAPIKey      string
}

func Load() (*Config, error) {
	baseURL := getEnv("MULTICA_BASE_URL", "")
	if baseURL == "" {
		return nil, fmt.Errorf("MULTICA_BASE_URL is required")
	}

	token := getEnv("MULTICA_TOKEN", "")
	if token == "" {
		return nil, fmt.Errorf("MULTICA_TOKEN is required")
	}

	transport := getEnv("MCP_TRANSPORT", "stdio")
	if transport != "stdio" && transport != "http" {
		return nil, fmt.Errorf("MCP_TRANSPORT must be 'stdio' or 'http', got %q", transport)
	}

	httpPort := getEnvInt("MCP_HTTP_PORT", 8080)
	logLevel := getEnv("LOG_LEVEL", "info")
	readOnly := getEnvBool("MULTICA_READ_ONLY", false)
	apiKey := getEnv("MCP_API_KEY", "")

	return &Config{
		MulticaBaseURL: baseURL,
		MulticaToken:   token,
		MCPTransport:   transport,
		LogLevel:       logLevel,
		HTTPPort:       httpPort,
		ReadOnly:       readOnly,
		MCPAPIKey:      apiKey,
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
