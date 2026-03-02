package config

import "testing"

func TestLoad_ValidConfig(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("EMAIL_SEND_TIME", "10:00AM")
	t.Setenv("EMAIL_SEND_LIMIT", "5")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000, https://app.example.com")
	t.Setenv("CORS_ALLOW_CREDENTIALS", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.EmailSendLimit != 5 {
		t.Fatalf("expected EmailSendLimit=5, got %d", cfg.EmailSendLimit)
	}
	if cfg.EmailSendTime != "10:00AM" {
		t.Fatalf("expected EmailSendTime=10:00AM, got %s", cfg.EmailSendTime)
	}
	if len(cfg.CORSAllowedOrigins) != 2 {
		t.Fatalf("expected 2 CORS origins, got %d", len(cfg.CORSAllowedOrigins))
	}
	if !cfg.CORSAllowCredentials {
		t.Fatal("expected CORSAllowCredentials=true")
	}
}

func TestLoad_InvalidEmailSendTime(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("EMAIL_SEND_TIME", "25:61")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid EMAIL_SEND_TIME")
	}
}

func TestLoad_InvalidEmailSendLimit(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("EMAIL_SEND_LIMIT", "0")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid EMAIL_SEND_LIMIT")
	}
}

func TestLoad_MissingRequiredVariable(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("DATABASE_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
}

func setRequiredEnv(t *testing.T) {
	t.Helper()

	t.Setenv("APP_ENV", "test")
	t.Setenv("SERVER_PORT", "8080")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	t.Setenv("UNOSEND_API_KEY", "test-api-key")
	t.Setenv("EMAIL_SEND_TIME", "10:00AM")
	t.Setenv("EMAIL_SEND_LIMIT", "3")
	t.Setenv("CLERK_WEBHOOK_SECRET", "whsec_test")
	t.Setenv("CLERK_JWKS_URL", "https://clerk.test/jwks")
	t.Setenv("CLERK_ISSUER", "https://clerk.test")
}
