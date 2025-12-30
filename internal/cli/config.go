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
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			// Don't exit, just don't load config from home
		} else {
			// Search config in home directory with name ".scriberr" (without extension).
			viper.AddConfigPath(home)
			viper.SetConfigType("yaml")
			viper.SetConfigName(".scriberr")
		}
	}

	viper.SetEnvPrefix("SCRIBERR")
	viper.AutomaticEnv()

	// Try to read config, ignore error if not found
	_ = viper.ReadInConfig()
}

// SaveConfig saves the configuration to ~/.scriberr.yaml and returns the path
func SaveConfig(serverURL, token, watchFolder string) (string, error) {
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
		return "", err
	}
	configPath := filepath.Join(home, ".scriberr.yaml")
	return configPath, viper.WriteConfigAs(configPath)
}

// GetConfig returns the current configuration
func GetConfig() *Config {
	return &Config{
		ServerURL:   viper.GetString("server_url"),
		Token:       viper.GetString("token"),
		WatchFolder: viper.GetString("watch_folder"),
	}
}
