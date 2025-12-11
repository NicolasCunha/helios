// Package config handles environment-based configuration for Helios.
package config

import (
	"errors"
	"log"
	"os"
	"strconv"
	"time"
)

// Config represents the complete Helios configuration loaded from environment variables.
type Config struct {
	Server       ServerConfig
	Database     DatabaseConfig
	Docker       DockerConfig
	HealthCheck  HealthCheckConfig
	LogRetention LogRetentionConfig
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	Host string
	Port string
	Mode string // "debug" or "release"
}

// DatabaseConfig contains database settings.
type DatabaseConfig struct {
	Path string
}

// DockerConfig contains Docker daemon settings.
type DockerConfig struct {
	Host string
}

// HealthCheckConfig contains health check monitoring settings.
type HealthCheckConfig struct {
	Interval        time.Duration
	CPUThreshold    float64
	MemoryThreshold float64
	Enabled         bool
}

// LogRetentionConfig contains log retention settings.
type LogRetentionConfig struct {
	Days int
}

// Load reads configuration from environment variables with sensible defaults.
// All environment variables use the HELIOS_ prefix.
//
// Configuration variables:
//   - HELIOS_SERVER_HOST (default: "0.0.0.0")
//   - HELIOS_SERVER_MODE (default: "debug")
//   - HELIOS_DB_PATH (default: "/app/data/helios.db" or "./helios.db")
//   - HELIOS_DOCKER_HOST (default: "unix:///var/run/docker.sock")
//   - HELIOS_HEALTH_CHECK_ENABLED (default: "true")
//   - HELIOS_HEALTH_CHECK_INTERVAL (default: "30s")
//   - HELIOS_CPU_THRESHOLD (default: "90")
//   - HELIOS_MEMORY_THRESHOLD (default: "90")
//   - HELIOS_LOG_RETENTION_DAYS (default: "30")
//
// Returns an error if validation fails.
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host: getEnv("HELIOS_SERVER_HOST", "0.0.0.0"),
			Port: getEnv("HELIOS_SERVER_PORT", "8080"),
			Mode: getEnv("HELIOS_SERVER_MODE", "debug"),
		},
		Database: DatabaseConfig{
			Path: getDBPath(),
		},
		Docker: DockerConfig{
			Host: getEnv("HELIOS_DOCKER_HOST", "unix:///var/run/docker.sock"),
		},
		HealthCheck: HealthCheckConfig{
			Enabled:         getEnvBool("HELIOS_HEALTH_CHECK_ENABLED", true),
			Interval:        getEnvDuration("HELIOS_HEALTH_CHECK_INTERVAL", 30*time.Second),
			CPUThreshold:    getEnvFloat("HELIOS_CPU_THRESHOLD", 90.0),
			MemoryThreshold: getEnvFloat("HELIOS_MEMORY_THRESHOLD", 90.0),
		},
		LogRetention: LogRetentionConfig{
			Days: getEnvInt("HELIOS_LOG_RETENTION_DAYS", 30),
		},
	}

	// Validate configuration
	if err := validate(cfg); err != nil {
		log.Printf("Configuration validation failed: %v", err)
		return nil, errors.New("invalid configuration")
	}

	// Log loaded configuration
	log.Printf("Configuration loaded:")
	log.Printf("  Server: %s:%s (mode: %s)", cfg.Server.Host, cfg.Server.Port, cfg.Server.Mode)
	log.Printf("  Database: %s", cfg.Database.Path)
	log.Printf("  Docker Host: %s", cfg.Docker.Host)
	log.Printf("  Health Checks: enabled=%v, interval=%v, cpu_threshold=%.0f%%, memory_threshold=%.0f%%",
		cfg.HealthCheck.Enabled, cfg.HealthCheck.Interval,
		cfg.HealthCheck.CPUThreshold, cfg.HealthCheck.MemoryThreshold)
	log.Printf("  Log Retention: %d days", cfg.LogRetention.Days)

	return cfg, nil
}

// validate checks if the configuration is valid.
func validate(cfg *Config) error {
	// Validate thresholds
	if cfg.HealthCheck.CPUThreshold < 0 || cfg.HealthCheck.CPUThreshold > 100 {
		return errors.New("CPU threshold must be between 0 and 100")
	}
	if cfg.HealthCheck.MemoryThreshold < 0 || cfg.HealthCheck.MemoryThreshold > 100 {
		return errors.New("memory threshold must be between 0 and 100")
	}
	if cfg.HealthCheck.Interval < time.Second {
		return errors.New("health check interval must be at least 1 second")
	}
	if cfg.LogRetention.Days < 1 {
		return errors.New("log retention days must be at least 1")
	}

	return nil
}

// getDBPath determines the database path based on environment and filesystem.
// Priority:
//  1. HELIOS_DB_PATH environment variable
//  2. /app/data/helios.db (if /app/data exists - Docker container)
//  3. ./helios.db (development fallback)
func getDBPath() string {
	// Check environment variable first
	if path := os.Getenv("HELIOS_DB_PATH"); path != "" {
		return path
	}

	// Check if running in container
	if _, err := os.Stat("/app/data"); err == nil {
		return "/app/data/helios.db"
	}

	// Development fallback
	return "./helios.db"
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt retrieves an integer environment variable or returns a default value.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
		log.Printf("Warning: invalid integer value for %s: %s, using default: %d", key, value, defaultValue)
	}
	return defaultValue
}

// getEnvFloat retrieves a float environment variable or returns a default value.
func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
		log.Printf("Warning: invalid float value for %s: %s, using default: %.2f", key, value, defaultValue)
	}
	return defaultValue
}

// getEnvBool retrieves a boolean environment variable or returns a default value.
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
		log.Printf("Warning: invalid boolean value for %s: %s, using default: %v", key, value, defaultValue)
	}
	return defaultValue
}

// getEnvDuration retrieves a duration environment variable or returns a default value.
// Accepts values like "30s", "5m", "1h"
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		log.Printf("Warning: invalid duration value for %s: %s, using default: %v", key, value, defaultValue)
	}
	return defaultValue
}
