package cli

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	installCmd = &cobra.Command{
		Use:   "install [folder]",
		Short: "Install the watcher as a background service",
		Args:  cobra.MaximumNArgs(1),
		Run:   runInstall,
	}

	startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the watcher service",
		Run:   runStart,
	}

	stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop the watcher service",
		Run:   runStop,
	}

	uninstallCmd = &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the watcher service",
		Run:   runUninstall,
	}

	logsCmd = &cobra.Command{
		Use:   "logs",
		Short: "Tail the service logs",
		Run:   runLogs,
	}
)

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(logsCmd)
}

type program struct{}

func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	go p.run()
	return nil
}

func (p *program) run() {
	// Setup logging to file
	if err := setupServiceLogging(); err != nil {
		// Fallback to standard log if file logging fails
		log.Printf("Failed to setup file logging: %v", err)
	}

	log.Println("Service starting...")

	// This is the actual work the service does
	config := GetConfig()
	log.Printf("Loaded config: ServerURL=%s, WatchFolder=%s, TokenSet=%v", config.ServerURL, config.WatchFolder, config.Token != "")

	if config.WatchFolder == "" {
		log.Println("No watch folder configured. Please run 'scriberr install [folder]' first.")
		return
	}

	// Re-use the watch logic
	// We'll call a new function watchFolder(folder string)
	watchFolder(config.WatchFolder)
}

func (p *program) Stop(s service.Service) error {
	log.Println("Service stopping...")
	// Stop should not block. Return with a few seconds.
	return nil
}

func getServiceConfig(configPath string) *service.Config {
	// Get absolute path to executable
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	args := []string{"service-run"}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}

	return &service.Config{
		Name:        "scriberr-watcher",
		DisplayName: "Scriberr Watcher Service",
		Description: "Watches a folder and uploads audio files to Scriberr.",
		Executable:  ex,
		Arguments:   args,
	}
}

// Special hidden command that the service manager runs
var serviceRunCmd = &cobra.Command{
	Use:    "service-run",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		// Setup logging immediately to capture startup errors
		if err := setupServiceLogging(); err != nil {
			log.Printf("Failed to setup file logging: %v", err)
		}
		log.Println("Starting service-run command...")

		// Note: InitConfig has already run by now (via cobra.OnInitialize)
		// If --config was passed, viper is already using it.

		prg := &program{}
		// We don't need to pass configPath here because we are already running inside the service
		// and the arguments have already been parsed by cobra to set cfgFile.
		s, err := service.New(prg, getServiceConfig(""))
		if err != nil {
			log.Fatalf("Failed to create service: %v", err)
		}

		// Setup system logger
		logger, err := s.Logger(nil)
		if err != nil {
			log.Printf("Failed to get system logger: %v", err)
		} else {
			_ = logger.Info("Scriberr service starting...")
		}

		if err = s.Run(); err != nil {
			if logger != nil {
				_ = logger.Error(err)
			}
			log.Fatalf("Service failed to run: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(serviceRunCmd)
}

func runInstall(cmd *cobra.Command, args []string) {
	// If folder provided, save it
	var configPath string
	if len(args) > 0 {
		folder := args[0]
		absPath, err := filepath.Abs(folder)
		if err != nil {
			log.Fatalf("Failed to get absolute path: %v", err)
		}

		// Get current config values
		token := viper.GetString("token")
		serverURL := viper.GetString("server_url")

		// If running as root (sudo), try to inherit config from the original user
		if os.Geteuid() == 0 {
			sudoUser := os.Getenv("SUDO_USER")
			if sudoUser != "" {
				if u, err := user.Lookup(sudoUser); err == nil {
					userConfigPath := filepath.Join(u.HomeDir, ".scriberr.yaml")
					if _, err := os.Stat(userConfigPath); err == nil {
						// Read user config using a separate viper instance
						v := viper.New()
						v.SetConfigFile(userConfigPath)
						if err := v.ReadInConfig(); err == nil {
							if userToken := v.GetString("token"); userToken != "" {
								token = userToken
								fmt.Printf("Inherited token from user %s\n", sudoUser)
							}
							if userURL := v.GetString("server_url"); userURL != "" {
								serverURL = userURL
								fmt.Printf("Inherited server URL from user %s\n", sudoUser)
							}
						}
					}
				}
			}
		}

		var errSave error
		configPath, errSave = SaveConfig(serverURL, token, absPath)
		if errSave != nil {
			log.Fatalf("Failed to save config: %v", errSave)
		}
		fmt.Printf("Configured to watch: %s\n", absPath)
	} else {
		// Check if already configured
		config := GetConfig()
		if config.WatchFolder == "" {
			log.Fatalf("No watch folder specified. Usage: scriberr install [folder]")
		}
		// We need to know where the config is to pass it to the service.
		// Since we didn't save it, we assume it's in the default location or cfgFile.
		if cfgFile != "" {
			configPath = cfgFile
		} else {
			home, err := os.UserHomeDir()
			if err == nil {
				configPath = filepath.Join(home, ".scriberr.yaml")
			}
		}
	}

	s, err := service.New(&program{}, getServiceConfig(configPath))
	if err != nil {
		log.Fatal(err)
	}

	if err = s.Install(); err != nil {
		log.Fatalf("Failed to install service: %v", err)
	}
	fmt.Println("Service installed successfully.")
}

func runStart(cmd *cobra.Command, args []string) {
	s, err := service.New(&program{}, getServiceConfig(""))
	if err != nil {
		log.Fatal(err)
	}
	if err = s.Start(); err != nil {
		log.Fatalf("Failed to start service: %v", err)
	}
	fmt.Println("Service started.")
}

func runStop(cmd *cobra.Command, args []string) {
	s, err := service.New(&program{}, getServiceConfig(""))
	if err != nil {
		log.Fatal(err)
	}
	if err = s.Stop(); err != nil {
		log.Fatalf("Failed to stop service: %v", err)
	}
	fmt.Println("Service stopped.")
}

func runUninstall(cmd *cobra.Command, args []string) {
	s, err := service.New(&program{}, getServiceConfig(""))
	if err != nil {
		log.Fatal(err)
	}
	if err = s.Uninstall(); err != nil {
		log.Fatalf("Failed to uninstall service: %v", err)
	}
	fmt.Println("Service uninstalled.")
}

func getLogFilePath() string {
	// Use /tmp/scriberr-service.log for simplicity and broad access
	return "/tmp/scriberr-service.log"
}

func setupServiceLogging() error {
	logFile := getLogFilePath()
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	log.SetOutput(f)
	return nil
}

func runLogs(cmd *cobra.Command, args []string) {
	logFile := getLogFilePath()
	fmt.Printf("Tailing logs from %s...\n", logFile)

	// Use 'tail -f' to follow logs
	// This works on macOS/Linux. Windows might need a different approach or just cat.
	c := exec.Command("tail", "-f", logFile)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		fmt.Printf("Error tailing logs: %v\n", err)
	}
}
