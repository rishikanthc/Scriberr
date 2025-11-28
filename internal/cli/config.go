package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the CLI configuration
type Config struct {
	ServerURL   string `mapstructure:"server_url"`
	Token       string `mapstructure:"token"`
	WatchFolder string `mapstructure:"watch_folder"`
}

// InitConfig initializes the configuration
func InitConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	viper.AddConfigPath(home)
	viper.SetConfigType("yaml")
	viper.SetConfigName(".scriberr")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		// Config file found and loaded
	}
}

// SaveConfig saves the configuration to ~/.scriberr.yaml
func SaveConfig(serverURL, token, watchFolder string) error {
	if serverURL != "" {
		viper.Set("server_url", serverURL)
	}
	if token != "" {
		viper.Set("token", token)
	}
	if watchFolder != "" {
		viper.Set("watch_folder", watchFolder)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(home, ".scriberr.yaml")
	return viper.WriteConfigAs(configPath)
}

// GetConfig returns the current configuration
func GetConfig() *Config {
	return &Config{
		ServerURL:   viper.GetString("server_url"),
		Token:       viper.GetString("token"),
		WatchFolder: viper.GetString("watch_folder"),
	}
}
