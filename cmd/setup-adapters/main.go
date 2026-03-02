package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"scriberr/internal/config"
	"scriberr/internal/transcription/registry"
	"scriberr/pkg/logger"
)

func main() {
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	logger.Init(*logLevel)
	logger.Info("Starting adapter environment setup")

	cfg := config.Load()

	// Register all standard adapters
	registry.RegisterStandardAdapters(cfg)

	// Initialize all registered models synchronously
	ctx := context.Background()
	err := registry.GetRegistry().InitializeModelsSync(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during adapter setup: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Adapter environment setup completed successfully")
}
