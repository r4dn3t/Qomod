package config

import (
	"os"
	"strconv"
	"time"

	"qomod/services/users/internal/event/rabbitmq"
	"qomod/services/users/internal/repository/postgres"
)

// Config represents all application configuration loaded from environment variables.
type Config struct {
	AppEnv        string
	HTTPPort      string
	LogLevel      string
	RunMigrations bool

	Postgres postgres.Config
	RabbitMQ rabbitmq.Config
}

// Load reads and parses configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	return &Config{
		AppEnv:        getEnv("APP_ENV", "development"),
		HTTPPort:      getEnv("HTTP_PORT", "8081"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		RunMigrations: getEnvBool("RUN_MIGRATIONS", true),

		Postgres: postgres.Config{
			Host:            getEnv("POSTGRES_HOST", "localhost"),
			Port:            getEnv("POSTGRES_PORT", "5432"),
			User:            getEnv("POSTGRES_USER", "postgres"),
			Password:        getEnv("POSTGRES_PASSWORD", "postgres"),
			DBName:          getEnv("POSTGRES_DB", "qomod_users"),
			SSLMode:         getEnv("POSTGRES_SSLMODE", "disable"),
			MaxConns:        int32(getEnvInt("POSTGRES_MAX_CONNS", 25)),
			MinConns:        int32(getEnvInt("POSTGRES_MIN_CONNS", 5)),
			MaxConnLifetime: time.Hour,
			MaxConnIdleTime: 30 * time.Minute,
		},

		RabbitMQ: rabbitmq.Config{
			URL:          getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
			ExchangeName: getEnv("RABBITMQ_EXCHANGE", "users.events"),
		},
	}, nil
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}
