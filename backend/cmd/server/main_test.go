package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadOrCreateInstallTokenPersistsPrivateToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "install-token")
	first, err := loadOrCreateInstallToken(path)
	if err != nil {
		t.Fatalf("create install token: %v", err)
	}
	if len(first) != 64 {
		t.Fatalf("token length = %d, want 64 hex characters", len(first))
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat install token: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("token file mode = %o, want 600", info.Mode().Perm())
	}
	second, err := loadOrCreateInstallToken(path)
	if err != nil {
		t.Fatalf("reload install token: %v", err)
	}
	if first != second {
		t.Fatal("install token changed across restarts")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != first {
		t.Fatal("persisted token does not match returned token")
	}
}
