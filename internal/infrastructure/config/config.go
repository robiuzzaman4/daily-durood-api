package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv               string
	ServerPort           string
	DatabaseURL          string
	UnosendAPIKey        string
	EmailSendTime        string
	EmailSendLimit       int
	CORSAllowedOrigins   []string
	CORSAllowedMethods   []string
	CORSAllowedHeaders   []string
	CORSAllowCredentials bool
	ClerkWebhookSecret   string
	ClerkJWKSURL         string
	ClerkIssuer          string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load .env file: %w", err)
	}

	cfg := &Config{
		AppEnv:     getOrDefault("APP_ENV", "development"),
		ServerPort: getOrDefault("SERVER_PORT", "8080"),
	}

	var err error

	if cfg.DatabaseURL, err = required("DATABASE_URL"); err != nil {
		return nil, err
	}
	if cfg.UnosendAPIKey, err = required("UNOSEND_API_KEY"); err != nil {
		return nil, err
	}
	if cfg.EmailSendTime, err = required("EMAIL_SEND_TIME"); err != nil {
		return nil, err
	}
	if _, err = parseEmailSendTime(cfg.EmailSendTime); err != nil {
		return nil, err
	}

	sendLimitRaw, err := required("EMAIL_SEND_LIMIT")
	if err != nil {
		return nil, err
	}
	cfg.EmailSendLimit, err = strconv.Atoi(sendLimitRaw)
	if err != nil || cfg.EmailSendLimit <= 0 {
		return nil, errors.New("EMAIL_SEND_LIMIT must be a positive integer")
	}
	if cfg.ClerkWebhookSecret, err = required("CLERK_WEBHOOK_SECRET"); err != nil {
		return nil, err
	}
	if cfg.ClerkJWKSURL, err = required("CLERK_JWKS_URL"); err != nil {
		return nil, err
	}
	cfg.CORSAllowedOrigins = parseCSVOrDefault(os.Getenv("CORS_ALLOWED_ORIGINS"), []string{"*"})
	cfg.CORSAllowedMethods = parseCSVOrDefault(os.Getenv("CORS_ALLOWED_METHODS"), []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
	cfg.CORSAllowedHeaders = parseCSVOrDefault(os.Getenv("CORS_ALLOWED_HEADERS"), []string{"Authorization", "Content-Type", "X-Request-ID"})
	cfg.CORSAllowCredentials = strings.EqualFold(strings.TrimSpace(os.Getenv("CORS_ALLOW_CREDENTIALS")), "true")
	cfg.ClerkIssuer = strings.TrimSpace(os.Getenv("CLERK_ISSUER"))

	return cfg, nil
}

func parseEmailSendTime(raw string) (time.Time, error) {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(raw), " ", ""))
	parsed, err := time.Parse("3:04PM", normalized)
	if err != nil {
		return time.Time{}, fmt.Errorf("EMAIL_SEND_TIME must be in 12-hour format like 10:00AM: %w", err)
	}
	return parsed, nil
}

func required(key string) (string, error) {
	val, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(val) == "" {
		return "", fmt.Errorf("missing required environment variable: %s", key)
	}
	return strings.TrimSpace(val), nil
}

func getOrDefault(key, fallback string) string {
	val, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(val) == "" {
		return fallback
	}
	return strings.TrimSpace(val)
}

func parseCSVOrDefault(raw string, fallback []string) []string {
	items := strings.Split(raw, ",")
	values := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		values = append(values, trimmed)
	}

	if len(values) == 0 {
		return fallback
	}
	return values
}
