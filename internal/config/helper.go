package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// getEnv reads an environment variable or returns a default value.
func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultVal
	}
	return val
}

// getEnvAsInt reads an environment variable as int or returns a default.
func getEnvAsInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}

// getEnvAsBool reads an environment variable as bool or returns a default.
func getEnvAsBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return b
}

// getEnvAsDuration reads an environment variable as time.Duration or returns a default.
func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	val := os.Getenv(key)
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return defaultVal
	}
	return d
}
