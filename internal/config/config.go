package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	Port        string
	DatabaseUrl string
	BaseUrl     string
}

func Load() (Config, error) {
	cfg := Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseUrl: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		BaseUrl:     getEnv("BASE_URL", "http://localhost:8080"),
	}
	if cfg.DatabaseUrl == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	u, err := url.Parse(cfg.BaseUrl)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return Config{}, fmt.Errorf("BASE_URL must be a valid url, got %q", cfg.BaseUrl)

	}
	return cfg, nil
}

func MustLoad() Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}

func getEnv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}
