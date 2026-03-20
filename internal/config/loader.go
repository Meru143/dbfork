// Package config loads dbfork configuration from disk and environment.
package config

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config contains the dbfork connection defaults.
type Config struct {
	Host     string `mapstructure:"default_host"`
	Port     int    `mapstructure:"default_port"`
	User     string `mapstructure:"default_user"`
	Password string `mapstructure:"default_password"`
	Database string `mapstructure:"default_database"`
}

// LoadConfig reads configuration from ~/.dbfork/config.toml and environment.
func LoadConfig() (Config, error) {
	currentUser, err := user.Current()
	if err != nil {
		return Config{}, err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.AddConfigPath(filepath.Join(homeDir, ".dbfork"))
	v.SetDefault("default_host", "localhost")
	v.SetDefault("default_port", 5432)
	v.SetDefault("default_user", currentUser.Username)
	v.SetDefault("default_password", "")
	v.SetDefault("default_database", "")
	_ = v.BindEnv("default_host", "DBFORK_HOST")
	_ = v.BindEnv("default_port", "DBFORK_PORT")
	_ = v.BindEnv("default_user", "DBFORK_USER")
	_ = v.BindEnv("default_password", "DBFORK_PASSWORD")
	_ = v.BindEnv("default_database", "DBFORK_DATABASE")

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return Config{}, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
