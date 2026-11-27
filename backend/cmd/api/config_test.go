package main

import "testing"

func TestLoadConfig_MissingDatabaseURL_ReturnsError(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	if _, err := loadConfig(); err == nil {
		t.Fatal("want error when DATABASE_URL is empty")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("APP_ENV", "")

	c, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if c.HTTPAddr != ":8080" || c.AppEnv != "development" {
		t.Errorf("defaults = %+v", c)
	}
}
