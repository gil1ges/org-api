package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppHost string
	AppPort int
	DBHost  string
	DBPort  int
	DBUser  string
	DBPass  string
	DBName  string
	DBSSL   string
}

func Load() Config {
	return Config{
		AppHost: getEnv("APP_HOST", "0.0.0.0"),
		AppPort: getEnvInt("APP_PORT", 8080),
		DBHost:  getEnv("DB_HOST", ""),
		DBPort:  getEnvInt("DB_PORT", 0),
		DBUser:  getEnv("DB_USER", ""),
		DBPass:  getEnv("DB_PASSWORD", ""),
		DBName:  getEnv("DB_NAME", ""),
		DBSSL:   getEnv("DB_SSLMODE", "disable"),
	}
}

func (c Config) Validate() error {
	missing := make([]string, 0, 5)
	if c.DBHost == "" {
		missing = append(missing, "DB_HOST")
	}
	if c.DBPort <= 0 {
		missing = append(missing, "DB_PORT")
	}
	if c.DBUser == "" {
		missing = append(missing, "DB_USER")
	}
	if c.DBPass == "" {
		missing = append(missing, "DB_PASSWORD")
	}
	if c.DBName == "" {
		missing = append(missing, "DB_NAME")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}

	return nil
}

func (c Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost,
		c.DBPort,
		c.DBUser,
		c.DBPass,
		c.DBName,
		c.DBSSL,
	)
}

func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.AppHost, c.AppPort)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func getEnvInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
