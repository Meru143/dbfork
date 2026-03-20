package config

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"
)

func TestLoadConfigReadsTOMLFixture(t *testing.T) {
	homeDir := t.TempDir()
	setHomeDir(t, homeDir)

	configDir := filepath.Join(homeDir, ".dbfork")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.toml")
	configBody := `default_host = "db.local"
default_port = 6543
default_user = "alice"
default_password = "secret"
default_database = "myapp_development"
`
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	want := Config{
		Host:     "db.local",
		Port:     6543,
		User:     "alice",
		Password: "secret",
		Database: "myapp_development",
	}
	if got != want {
		t.Fatalf("unexpected config: %+v", got)
	}
}

func TestLoadConfigEnvOverridesFileValue(t *testing.T) {
	homeDir := t.TempDir()
	setHomeDir(t, homeDir)
	t.Setenv("DBFORK_HOST", "override.local")

	configDir := filepath.Join(homeDir, ".dbfork")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(`default_host = "file.local"`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if got.Host != "override.local" {
		t.Fatalf("expected env override, got %+v", got)
	}
}

func TestLoadConfigUsesDBFORKConfigOverride(t *testing.T) {
	homeDir := t.TempDir()
	setHomeDir(t, homeDir)

	overridePath := filepath.Join(t.TempDir(), "dbfork.toml")
	t.Setenv("DBFORK_CONFIG", overridePath)

	configBody := `default_host = "override-file.local"
default_port = 6543
default_user = "override-user"
default_password = "secret"
default_database = "override_db"
`
	if err := os.WriteFile(overridePath, []byte(configBody), 0o600); err != nil {
		t.Fatalf("write override config: %v", err)
	}

	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if got.Host != "override-file.local" || got.Port != 6543 || got.User != "override-user" || got.Password != "secret" || got.Database != "override_db" {
		t.Fatalf("unexpected config override: %+v", got)
	}
}

func TestLoadConfigAppliesDefaultsWhenNoConfigExists(t *testing.T) {
	homeDir := t.TempDir()
	setHomeDir(t, homeDir)

	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	currentUser, err := user.Current()
	if err != nil {
		t.Fatalf("current user: %v", err)
	}

	if got.Host != "localhost" {
		t.Fatalf("expected default host, got %+v", got)
	}
	if got.Port != 5432 {
		t.Fatalf("expected default port, got %+v", got)
	}
	if got.User != currentUser.Username {
		t.Fatalf("expected default user %q, got %+v", currentUser.Username, got)
	}
	if got.Password != "" || got.Database != "" {
		t.Fatalf("expected empty password/database defaults, got %+v", got)
	}
}

func setHomeDir(t *testing.T, homeDir string) {
	t.Helper()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
}
