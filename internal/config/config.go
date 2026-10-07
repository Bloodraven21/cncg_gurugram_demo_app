package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr     string
	OTLPEndpoint string
	ServiceName  string
	LogLevel     string
}

func Load() Config {
	return Config{
		HTTPAddr:     getEnv("HTTP_ADDR", ":8080"),
		OTLPEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		ServiceName:  getEnv("OTEL_SERVICE_NAME", "o11y-starter"),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
