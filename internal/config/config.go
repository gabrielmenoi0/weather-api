package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName string
	AppEnv  string
	AppPort string

	StaticAPIToken string

	OpenMeteoBaseURL      string
	OpenMeteoGeocodingURL string
	HTTPTimeoutSeconds    int

	LogLevel  string
	LogFormat string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppName: getEnv("APP_NAME", "weather-api"),
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: getEnv("APP_PORT", "8080"),

		StaticAPIToken: getEnv("STATIC_API_TOKEN", ""),

		OpenMeteoBaseURL:      getEnv("OPEN_METEO_BASE_URL", "https://api.open-meteo.com/v1"),
		OpenMeteoGeocodingURL: getEnv("OPEN_METEO_GEOCODING_URL", "https://geocoding-api.open-meteo.com/v1"),
		HTTPTimeoutSeconds:    getEnvInt("HTTP_TIMEOUT_SECONDS", 10),

		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),
	}

	if cfg.StaticAPIToken == "" {
		return nil, fmt.Errorf("STATIC_API_TOKEN é obrigatório")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
