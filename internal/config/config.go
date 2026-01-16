package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort string
	BaseURL string

	DatabaseURL   string
	MigrationsDir string

	ClickQueueSize     int
	ClickWorkers       int
	ClickBatchSize     int
	ClickFlushInterval time.Duration
	ClickWriteTimeout  time.Duration

	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration
	HTTPIdleTimeout  time.Duration

	ShutdownTimeout time.Duration
}

func MustLoad() Config {
	// Подхватываем .env, если он есть (не ошибка, если нет)
	_ = godotenv.Load()

	c := Config{
		AppPort: "8080",
		BaseURL: "http://localhost:8080",

		MigrationsDir: "migrations",

		ClickQueueSize:     1024,
		ClickWorkers:       4,
		ClickBatchSize:     200,
		ClickFlushInterval: 1 * time.Second,
		ClickWriteTimeout:  2 * time.Second,

		HTTPReadTimeout:  5 * time.Second,
		HTTPWriteTimeout: 10 * time.Second,
		HTTPIdleTimeout:  60 * time.Second,

		ShutdownTimeout: 10 * time.Second,
	}

	if v := os.Getenv("APP_PORT"); v != "" {
		c.AppPort = v
	}
	if v := os.Getenv("BASE_URL"); v != "" {
		c.BaseURL = v
	}

	c.DatabaseURL = os.Getenv("DATABASE_URL")
	if c.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	if v := os.Getenv("MIGRATIONS_DIR"); v != "" {
		c.MigrationsDir = v
	}

	c.ClickQueueSize = getInt("CLICK_QUEUE_SIZE", c.ClickQueueSize)
	c.ClickWorkers = getInt("CLICK_WORKERS", c.ClickWorkers)
	c.ClickBatchSize = getInt("CLICK_BATCH_SIZE", c.ClickBatchSize)

	c.ClickFlushInterval = getDuration("CLICK_FLUSH_INTERVAL", c.ClickFlushInterval)
	c.ClickWriteTimeout = getDuration("CLICK_WRITE_TIMEOUT", c.ClickWriteTimeout)

	c.HTTPReadTimeout = getDuration("HTTP_READ_TIMEOUT", c.HTTPReadTimeout)
	c.HTTPWriteTimeout = getDuration("HTTP_WRITE_TIMEOUT", c.HTTPWriteTimeout)
	c.HTTPIdleTimeout = getDuration("HTTP_IDLE_TIMEOUT", c.HTTPIdleTimeout)

	c.ShutdownTimeout = getDuration("SHUTDOWN_TIMEOUT", c.ShutdownTimeout)

	return c
}

func (c Config) HTTPAddr() string {
	return ":" + c.AppPort
}

func getInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func getDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return def
	}
	return d
}
