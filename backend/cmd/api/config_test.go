package main

import (
	"strings"
	"testing"
	"time"
)

func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("JWT_SECRET", strings.Repeat("k", 32))
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("ACCESS_TOKEN_TTL", "")
}

func TestLoadConfig_MissingDatabaseURL_ReturnsError(t *testing.T) {
	setValidEnv(t)
	t.Setenv("DATABASE_URL", "")

	if _, err := loadConfig(); err == nil {
		t.Fatal("want error when DATABASE_URL is empty")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	setValidEnv(t)

	c, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if c.HTTPAddr != ":8080" || c.AppEnv != "development" || c.SMTPPort != 1025 || c.AccessTTL != 15*time.Minute {
		t.Errorf("defaults = HTTPAddr %q AppEnv %q SMTPPort %d AccessTTL %v", c.HTTPAddr, c.AppEnv, c.SMTPPort, c.AccessTTL)
	}
}

func TestLoadConfig_Invalid_ReturnsError_CF01(t *testing.T) {
	tests := []struct {
		name, key, value string
	}{
		{"thiếu JWT_SECRET", "JWT_SECRET", ""},
		{"JWT_SECRET quá ngắn", "JWT_SECRET", "short"},
		{"SMTP_PORT không phải số", "SMTP_PORT", "abc"},
		{"ACCESS_TOKEN_TTL âm", "ACCESS_TOKEN_TTL", "-5m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidEnv(t)
			t.Setenv(tt.key, tt.value)
			_, err := loadConfig()
			if err == nil {
				t.Fatal("want error")
			}
			if strings.Contains(err.Error(), strings.Repeat("k", 32)) {
				t.Error("thông báo lỗi lộ secret")
			}
		})
	}
}
