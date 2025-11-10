package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

// Config aggregates runtime configuration required by API/Bot processes.
type Config struct {
	ServerPort        string
	PostgresURL       string
	TelegramToken     string
	TelegramAPIBase   string
	ServiceAPIToken   string
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
}

// Load reads env vars and populates Config with sane defaults.
func Load() Config {
	return Config{
		ServerPort:        getEnv("SERVER_PORT", "8080"),
		PostgresURL:       getEnv("POSTGRES_URL", "postgres://tg_todo:change-me@localhost:5432/tg_todo?sslmode=disable"),
		TelegramToken:     getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramAPIBase:   getEnv("TELEGRAM_API_URL", "https://api.telegram.org"),
		ServiceAPIToken:   getEnv("SERVICE_API_TOKEN", ""),
		ReadTimeout:       getDuration("SERVER_READ_TIMEOUT", 15*time.Second),
		WriteTimeout:      getDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
		DBMaxOpenConns:    getInt("DB_MAX_OPEN_CONNS", 10),
		DBMaxIdleConns:    getInt("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLifetime: getDuration("DB_CONN_MAX_LIFETIME", 1800*time.Second),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	val := getEnv(key, "")
	if val == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		log.Printf("config: invalid duration for %s, using fallback %s", key, fallback)
		return fallback
	}
	return time.Duration(parsed) * time.Second
}

func getInt(key string, fallback int) int {
	val := getEnv(key, "")
	if val == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		log.Printf("config: invalid int for %s, using fallback %d", key, fallback)
		return fallback
	}
	return parsed
}
