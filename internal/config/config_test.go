package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("TG_GROUP_HTTP_PORT", "")
	t.Setenv("TG_GROUP_ENV", "")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q, want %q", got.HTTPAddr, ":8080")
	}
	if got.Environment != "development" {
		t.Fatalf("Environment = %q, want %q", got.Environment, "development")
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Setenv("TG_GROUP_HTTP_PORT", "70000")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid port error")
	}
}
