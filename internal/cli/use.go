// Package cli provides the dbfork use command.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Meru143/dbfork/internal/config"
	"github.com/Meru143/dbfork/internal/postgres"
	"github.com/Meru143/dbfork/internal/state"
)

var useCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Mark a branch as active in the current directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUse(cmd.Context(), useDeps{
			loadConfig: func() (config.Config, error) {
				return config.LoadConfig()
			},
			loadState: func() (*state.State, error) {
				return state.Load()
			},
			getwd: func() (string, error) {
				return os.Getwd()
			},
			writeFile: func(path string, data []byte, perm os.FileMode) error {
				return os.WriteFile(path, data, perm)
			},
			output: cmd.OutOrStdout(),
		}, args[0])
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
}

type useDeps struct {
	loadConfig func() (config.Config, error)
	loadState  func() (*state.State, error)
	getwd      func() (string, error)
	writeFile  func(string, []byte, os.FileMode) error
	output     io.Writer
}

func runUse(_ context.Context, deps useDeps, name string) error {
	cfg, err := deps.loadConfig()
	if err != nil {
		return err
	}

	branches, err := deps.loadState()
	if err != nil {
		return err
	}

	branch, ok := findBranchByDisplayName(branches, name)
	if !ok {
		return fmt.Errorf("Branch '%s' not found.", name)
	}

	workingDir, err := deps.getwd()
	if err != nil {
		return err
	}

	activeFile := filepath.Join(workingDir, ".dbfork")
	if err := deps.writeFile(activeFile, []byte(branch.Name+"\n"), 0o644); err != nil {
		return err
	}

	dsnCfg := config.Config{
		Host:     branch.Host,
		Port:     branch.Port,
		User:     branch.User,
		Password: cfg.Password,
		Database: branch.Database,
	}
	if dsnCfg.Host == "" {
		dsnCfg.Host = cfg.Host
	}
	if dsnCfg.Port == 0 {
		dsnCfg.Port = cfg.Port
	}
	if dsnCfg.User == "" {
		dsnCfg.User = cfg.User
	}

	dsn := postgres.BuildDSN(dsnCfg)
	_, _ = fmt.Fprintf(deps.output, "✓ Switched to branch '%s'\n", branch.Name)
	_, _ = fmt.Fprintf(deps.output, "  DATABASE_URL=%s\n", dsn)
	return nil
}
