package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	WasenderAPIKey        string
	WasenderWebhookSecret string
	DatabaseURL           string
	BotPhone              string
	Port                  string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		WasenderAPIKey:        os.Getenv("WASENDER_API_KEY"),
		WasenderWebhookSecret: os.Getenv("WASENDER_WEBHOOK_SECRET"),
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		BotPhone:              os.Getenv("BOT_PHONE"),
		Port:                  os.Getenv("PORT"),
	}

	if cfg.WasenderAPIKey == "" {
		return nil, fmt.Errorf("WASENDER_API_KEY is required")
	}
	if cfg.WasenderWebhookSecret == "" {
		return nil, fmt.Errorf("WASENDER_WEBHOOK_SECRET is required")
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.BotPhone == "" {
		return nil, fmt.Errorf("BOT_PHONE is required")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	return cfg, nil
}
