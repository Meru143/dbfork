package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestLoadRootConfigReadsOverrideFile(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	configPath := filepath.Join(t.TempDir(), "dbfork.toml")
	configBody := `default_host = "config.local"
default_port = 6543
default_user = "config-user"
default_password = "config-pass"
default_database = "config-db"
`
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("DBFORK_CONFIG", configPath)

	cmd := newRootConfigTestCommand(t)
	if err := loadRootConfig(cmd); err != nil {
		t.Fatalf("load root config: %v", err)
	}

	if viper.GetString("default_host") != "config.local" || viper.GetInt("default_port") != 6543 || viper.GetString("default_user") != "config-user" || viper.GetString("default_password") != "config-pass" || viper.GetString("default_database") != "config-db" {
		t.Fatalf("unexpected viper config values: host=%q port=%d user=%q db=%q", viper.GetString("default_host"), viper.GetInt("default_port"), viper.GetString("default_user"), viper.GetString("default_database"))
	}
}

func TestLoadRootConfigFlagsOverrideLoadedConfig(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	configPath := filepath.Join(t.TempDir(), "dbfork.toml")
	if err := os.WriteFile(configPath, []byte(`default_host = "config.local"`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("DBFORK_CONFIG", configPath)

	cmd := newRootConfigTestCommand(t)
	if err := cmd.Flags().Set("host", "flag.local"); err != nil {
		t.Fatalf("set host flag: %v", err)
	}
	if err := cmd.Flags().Set("port", "7654"); err != nil {
		t.Fatalf("set port flag: %v", err)
	}
	if err := cmd.Flags().Set("user", "flag-user"); err != nil {
		t.Fatalf("set user flag: %v", err)
	}
	if err := cmd.Flags().Set("password", "flag-pass"); err != nil {
		t.Fatalf("set password flag: %v", err)
	}
	if err := cmd.Flags().Set("source", "flag-db"); err != nil {
		t.Fatalf("set source flag: %v", err)
	}
	if err := cmd.Flags().Set("verbose", "true"); err != nil {
		t.Fatalf("set verbose flag: %v", err)
	}

	if err := loadRootConfig(cmd); err != nil {
		t.Fatalf("load root config: %v", err)
	}

	if viper.GetString("default_host") != "flag.local" || viper.GetInt("default_port") != 7654 || viper.GetString("default_user") != "flag-user" || viper.GetString("default_password") != "flag-pass" || viper.GetString("default_database") != "flag-db" || !viper.GetBool("verbose") {
		t.Fatalf("unexpected overridden config values: host=%q port=%d user=%q db=%q verbose=%v", viper.GetString("default_host"), viper.GetInt("default_port"), viper.GetString("default_user"), viper.GetString("default_database"), viper.GetBool("verbose"))
	}
}

func newRootConfigTestCommand(t *testing.T) *cobra.Command {
	t.Helper()

	cmd := &cobra.Command{Use: "test"}
	flags := cmd.Flags()
	flags.String("host", "localhost", "")
	flags.Int("port", 5432, "")
	flags.String("user", "", "")
	flags.String("password", "", "")
	flags.String("source", "", "")
	flags.Bool("verbose", false, "")
	return cmd
}
