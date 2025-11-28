package cli

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
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
)

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(uninstallCmd)
}

type program struct{}

func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	go p.run()
	return nil
}

func (p *program) run() {
	// This is the actual work the service does
	config := GetConfig()
	if config.WatchFolder == "" {
		log.Println("No watch folder configured. Please run 'scriberr install [folder]' first.")
		return
	}

	// Re-use the watch logic
	// We need to mock the command structure or extract the logic
	// For now, we'll just call the logic directly if we extract it,
	// but since it's in the same package, we can just call a function.
	// However, runWatch expects *cobra.Command.
	// Let's refactor runWatch to separate the logic.

	// We'll call a new function watchFolder(folder string)
	watchFolder(config.WatchFolder)
}

func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	return nil
}

func getServiceConfig() *service.Config {
	// Get absolute path to executable
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	return &service.Config{
		Name:        "scriberr-watcher",
		DisplayName: "Scriberr Watcher Service",
		Description: "Watches a folder and uploads audio files to Scriberr.",
		Executable:  ex,
		Arguments:   []string{"service-run"}, // Special hidden command to run the service logic
	}
}

// Special hidden command that the service manager runs
var serviceRunCmd = &cobra.Command{
	Use:    "service-run",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		prg := &program{}
		s, err := service.New(prg, getServiceConfig())
		if err != nil {
			log.Fatal(err)
		}
		if err = s.Run(); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(serviceRunCmd)
}

func runInstall(cmd *cobra.Command, args []string) {
	// If folder provided, save it
	if len(args) > 0 {
		folder := args[0]
		absPath, err := filepath.Abs(folder)
		if err != nil {
			log.Fatalf("Failed to get absolute path: %v", err)
		}

		if err := SaveConfig("", "", absPath); err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}
		fmt.Printf("Configured to watch: %s\n", absPath)
	} else {
		// Check if already configured
		config := GetConfig()
		if config.WatchFolder == "" {
			log.Fatalf("No watch folder specified. Usage: scriberr install [folder]")
		}
	}

	s, err := service.New(&program{}, getServiceConfig())
	if err != nil {
		log.Fatal(err)
	}

	if err = s.Install(); err != nil {
		log.Fatalf("Failed to install service: %v", err)
	}
	fmt.Println("Service installed successfully.")
}

func runStart(cmd *cobra.Command, args []string) {
	s, err := service.New(&program{}, getServiceConfig())
	if err != nil {
		log.Fatal(err)
	}
	if err = s.Start(); err != nil {
		log.Fatalf("Failed to start service: %v", err)
	}
	fmt.Println("Service started.")
}

func runStop(cmd *cobra.Command, args []string) {
	s, err := service.New(&program{}, getServiceConfig())
	if err != nil {
		log.Fatal(err)
	}
	if err = s.Stop(); err != nil {
		log.Fatalf("Failed to stop service: %v", err)
	}
	fmt.Println("Service stopped.")
}

func runUninstall(cmd *cobra.Command, args []string) {
	s, err := service.New(&program{}, getServiceConfig())
	if err != nil {
		log.Fatal(err)
	}
	if err = s.Uninstall(); err != nil {
		log.Fatalf("Failed to uninstall service: %v", err)
	}
	fmt.Println("Service uninstalled.")
}
