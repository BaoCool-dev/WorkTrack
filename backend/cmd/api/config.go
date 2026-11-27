package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultAccessTTL = 15 * time.Minute // B4
	minJWTSecretLen  = 32               // khớp security.MinSecretLength
)

// config đọc từ biến môi trường (xem .env.example ở gốc repo).
// Không bao giờ log giá trị JWTSecret hay SMTPPassword.
type config struct {
	AppEnv      string
	HTTPAddr    string
	DatabaseURL string
	CORSOrigin  string

	JWTSecret string
	AccessTTL time.Duration

	// AppBaseURL là địa chỉ frontend, dùng dựng link kích hoạt trong email.
	AppBaseURL string

	SMTPHost     string
	SMTPPort     int
	SMTPFrom     string
	SMTPUsername string
	SMTPPassword string
}

func loadConfig() (config, error) {
	c := config{
		AppEnv:       getenv("APP_ENV", "development"),
		HTTPAddr:     getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		CORSOrigin:   getenv("CORS_ORIGIN", "http://localhost:5173"),
		JWTSecret:    os.Getenv("JWT_SECRET"),
		AppBaseURL:   getenv("APP_BASE_URL", "http://localhost:5173"),
		SMTPHost:     getenv("SMTP_HOST", "localhost"),
		SMTPFrom:     getenv("SMTP_FROM", "WorkTrack <no-reply@worktrack.local>"),
		SMTPUsername: os.Getenv("SMTP_USERNAME"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
	}

	var errs []error
	if c.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if len(c.JWTSecret) < minJWTSecretLen {
		errs = append(errs, fmt.Errorf("JWT_SECRET is required and must be at least %d bytes", minJWTSecretLen))
	}

	port, err := strconv.Atoi(getenv("SMTP_PORT", "1025"))
	if err != nil || port <= 0 || port > 65535 {
		errs = append(errs, errors.New("SMTP_PORT must be a valid port"))
	}
	c.SMTPPort = port

	c.AccessTTL = defaultAccessTTL
	if v := os.Getenv("ACCESS_TOKEN_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			errs = append(errs, errors.New("ACCESS_TOKEN_TTL must be a positive duration, e.g. 15m"))
		}
		c.AccessTTL = d
	}

	if err := errors.Join(errs...); err != nil {
		return config{}, err
	}
	return c, nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
