// Package cli provides the dbfork init command.
package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/owner/dbfork/internal/config"
	"github.com/owner/dbfork/internal/postgres"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize dbfork configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		currentUser, err := user.Current()
		if err != nil {
			return err
		}

		return runInit(cmd.Context(), os.Stdin, cmd.OutOrStdout(), homeDir, currentUser.Username, func(ctx context.Context, cfg config.Config) error {
			conn, err := postgres.Connect(ctx, postgres.BuildDSN(cfg))
			if err != nil {
				return err
			}
			defer conn.Close(ctx)

			return postgres.Ping(ctx, conn)
		})
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(
	ctx context.Context,
	input io.Reader,
	output io.Writer,
	homeDir string,
	defaultUser string,
	testConnection func(context.Context, config.Config) error,
) error {
	configDir := filepath.Join(homeDir, ".dbfork")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.toml")
	if _, err := os.Stat(configPath); err == nil {
		_, _ = fmt.Fprintln(output, "Config already exists")
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	reader := bufio.NewReader(input)

	host, err := prompt(reader, output, "Host", "localhost")
	if err != nil {
		return err
	}

	portText, err := prompt(reader, output, "Port", "5432")
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return fmt.Errorf("invalid port %q: %w", portText, err)
	}

	username, err := prompt(reader, output, "User", defaultUser)
	if err != nil {
		return err
	}

	password, err := prompt(reader, output, "Password", "")
	if err != nil {
		return err
	}

	database, err := prompt(reader, output, "Database", "")
	if err != nil {
		return err
	}
	if database == "" {
		return fmt.Errorf("database name cannot be empty")
	}

	cfg := config.Config{
		Host:     host,
		Port:     port,
		User:     username,
		Password: password,
		Database: database,
	}

	if err := testConnection(ctx, cfg); err != nil {
		return err
	}

	if err := writeInitConfig(configPath, cfg); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(output, "Config written to %s\n", configPath)
	return nil
}

func prompt(reader *bufio.Reader, output io.Writer, label, defaultValue string) (string, error) {
	if defaultValue == "" {
		if _, err := fmt.Fprintf(output, "%s: ", label); err != nil {
			return "", err
		}
	} else {
		if _, err := fmt.Fprintf(output, "%s [%s]: ", label, defaultValue); err != nil {
			return "", err
		}
	}

	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}

	value := strings.TrimSpace(line)
	if value == "" {
		return defaultValue, nil
	}

	return value, nil
}

func writeInitConfig(configPath string, cfg config.Config) error {
	v := viper.New()
	v.Set("default_host", cfg.Host)
	v.Set("default_port", cfg.Port)
	v.Set("default_user", cfg.User)
	v.Set("default_password", cfg.Password)
	v.Set("default_database", cfg.Database)
	v.SetConfigType("toml")
	return v.WriteConfigAs(configPath)
}
