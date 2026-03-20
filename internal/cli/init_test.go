package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/owner/dbfork/internal/config"
)

func TestRunInitWritesConfigAfterSuccessfulConnection(t *testing.T) {
	tempHome := t.TempDir()
	input := strings.NewReader("\n\n\n\nmyapp_development\n")
	var output bytes.Buffer
	var connected config.Config

	err := runInit(
		context.Background(),
		input,
		&output,
		tempHome,
		"tester",
		func(_ context.Context, cfg config.Config) error {
			connected = cfg
			return nil
		},
	)
	if err != nil {
		t.Fatalf("run init: %v", err)
	}

	if connected.Host != "localhost" || connected.Port != 5432 || connected.User != "tester" || connected.Password != "" || connected.Database != "myapp_development" {
		t.Fatalf("unexpected config passed to connection test: %+v", connected)
	}

	configPath := filepath.Join(tempHome, ".dbfork", "config.toml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}

	configText := string(content)
	for _, want := range []string{
		`default_host = 'localhost'`,
		`default_port = 5432`,
		`default_user = 'tester'`,
		`default_database = 'myapp_development'`,
	} {
		if !strings.Contains(configText, want) {
			t.Fatalf("expected config to contain %q, got:\n%s", want, configText)
		}
	}

	if !strings.Contains(output.String(), configPath) {
		t.Fatalf("expected success output to mention %q, got %q", configPath, output.String())
	}
}

func TestRunInitReturnsWhenConfigAlreadyExists(t *testing.T) {
	tempHome := t.TempDir()
	configDir := filepath.Join(tempHome, ".dbfork")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configPath, []byte("default_host = 'localhost'\n"), 0o644); err != nil {
		t.Fatalf("write existing config: %v", err)
	}

	var output bytes.Buffer
	called := false
	err := runInit(
		context.Background(),
		strings.NewReader(""),
		&output,
		tempHome,
		"tester",
		func(_ context.Context, cfg config.Config) error {
			called = true
			return nil
		},
	)
	if err != nil {
		t.Fatalf("run init with existing config: %v", err)
	}

	if called {
		t.Fatal("expected connection test not to run when config already exists")
	}

	if !strings.Contains(output.String(), "Config already exists") {
		t.Fatalf("expected existing-config message, got %q", output.String())
	}
}

func TestInitCommandIsRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"init"})
	if err != nil {
		t.Fatalf("find init command: %v", err)
	}

	if cmd == nil || cmd.Use != "init" {
		t.Fatalf("expected init command to be registered, got %#v", cmd)
	}
}
