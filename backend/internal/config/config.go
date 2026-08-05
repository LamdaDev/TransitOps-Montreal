package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL    string
	Port           string
	DataProvider   string
	IngestInterval time.Duration
	STMAPIKey      string
	STMGtfsRTURL   string
}

func Load() Config {
	return Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://transitops:transitops@localhost:5432/transitops?sslmode=disable"),
		Port:           getEnv("PORT", "8080"),
		DataProvider:   getEnv("DATA_PROVIDER", "mock"),
		IngestInterval: getDurationEnv("INGEST_INTERVAL_SECONDS", 10*time.Second),
		STMAPIKey:      os.Getenv("STM_API_KEY"),
		STMGtfsRTURL:   os.Getenv("STM_GTFS_RT_VEHICLE_POSITIONS_URL"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return fallback
	}

	return time.Duration(seconds) * time.Second
}
