// Package main provides the CLI entry point for cc-connect.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/smy-101/cc-connect/internal/app"
	"github.com/smy-101/cc-connect/internal/core"
)

// Build information (injected via -ldflags)
var (
	version   = "dev"
	gitCommit = "unknown"
	buildDate = "unknown"
)

func main() {
	// Parse flags
	configPath := flag.String("config", "./config.toml", "path to configuration file")
	showVersion := flag.Bool("version", false, "show version information")
	flag.Parse()

	// Show version and exit
	if *showVersion {
		fmt.Printf("cc-connect %s\n", version)
		fmt.Printf("  Git commit: %s\n", gitCommit)
		fmt.Printf("  Build date: %s\n", buildDate)
		os.Exit(0)
	}

	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	config, err := loadConfig(*configPath)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err, "path", *configPath)
		os.Exit(1)
	}

	// Create application
	application, err := app.New(config)
	if err != nil {
		slog.Error("Failed to create application", "error", err)
		os.Exit(1)
	}

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start application
	slog.Info("Starting cc-connect", "version", version)
	slog.Info("Process initialized, waiting for Feishu long connection readiness")
	if err := application.Start(ctx); err != nil {
		slog.Error("Failed to start application", "error", err)
		os.Exit(1)
	}
	slog.Info("Feishu long connection ready", "next_step", "continue event subscription setup in Feishu Open Platform")

	// Wait for shutdown signal
	sig := <-sigChan
	slog.Info("Received shutdown signal", "signal", sig.String())

	// Graceful shutdown
	slog.Info("Shutting down gracefully...")
	if err := application.Stop(); err != nil {
		slog.Error("Error during shutdown", "error", err)
	}

	application.WaitForShutdown()
	slog.Info("Application stopped")
}

// loadConfig loads the configuration from the specified path.
func loadConfig(path string) (*core.AppConfig, error) {
	loader := core.NewTOMLLoader()
	config, err := loader.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Validate configuration
	validator := core.NewConfigValidator()
	if err := validator.Validate(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Log warnings
	warnings := validator.Warnings(config)
	for _, w := range warnings {
		slog.Warn("Configuration warning", "warning", w)
	}

	return config, nil
}
