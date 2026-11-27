package main

import (
	"errors"
	"os"
)

// config đọc từ biến môi trường (xem .env.example ở gốc repo).
type config struct {
	AppEnv      string
	HTTPAddr    string
	DatabaseURL string
	CORSOrigin  string
}

func loadConfig() (config, error) {
	c := config{
		AppEnv:      getenv("APP_ENV", "development"),
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		CORSOrigin:  getenv("CORS_ORIGIN", "http://localhost:5173"),
	}
	if c.DatabaseURL == "" {
		return config{}, errors.New("DATABASE_URL is required")
	}
	return c, nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
