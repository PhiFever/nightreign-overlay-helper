package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/PhiFever/nightreign-overlay-helper/internal/config"
	"github.com/PhiFever/nightreign-overlay-helper/internal/detector"
	"github.com/PhiFever/nightreign-overlay-helper/internal/logger"
	"github.com/PhiFever/nightreign-overlay-helper/internal/updater"
	"github.com/PhiFever/nightreign-overlay-helper/pkg/version"
)

func main() {
	fmt.Printf("Starting %s...\n", version.GetFullName())

	// Initialize logger
	if _, err := logger.Setup(logger.INFO); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Logger initialized")
	logger.Infof("Application: %s", version.GetFullName())
	logger.Infof("Version: %s", version.Version)
	logger.Infof("Author: %s", version.Author)

	// Load configuration
	cfg, err := config.Get()
	if err != nil {
		logger.Errorf("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	logger.Info("Configuration loaded successfully")
	logger.Debugf("Update interval: %.2f seconds", cfg.UpdateInterval)
	logger.Debugf("Time scale: %.2f", cfg.TimeScale)

	// Initialize detector registry
	registry := detector.NewDetectorRegistry()

	// Create and register detectors
	dayDetector := detector.NewDayDetector(cfg)
	registry.Register(dayDetector)

	// Initialize all detectors
	logger.Info("Initializing detectors...")
	if err := registry.InitializeAll(); err != nil {
		logger.Errorf("Failed to initialize detectors: %v", err)
		os.Exit(1)
	}
	logger.Info("All detectors initialized successfully")

	// Create updater
	upd := updater.NewUpdater(cfg, registry)

	// TODO: Initialize UI
	logger.Info("TODO: Initialize UI")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start updater
	logger.Info("Starting updater...")
	if err := upd.Start(ctx); err != nil {
		logger.Errorf("Failed to start updater: %v", err)
		os.Exit(1)
	}
	logger.Info("Updater started successfully")

	logger.Info("Application started successfully")
	logger.Info("Press Ctrl+C to exit")

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	// Accept SIGINT (Ctrl+C) and SIGTERM
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive a signal
	sig := <-quit
	logger.Infof("Received signal: %v", sig)
	logger.Info("Shutting down gracefully...")

	// Cancel context
	cancel()

	// Stop updater
	if err := upd.Stop(); err != nil {
		logger.Errorf("Error stopping updater: %v", err)
	}

	// Cleanup detectors
	logger.Info("Cleaning up detectors...")
	if err := registry.CleanupAll(); err != nil {
		logger.Errorf("Error cleaning up detectors: %v", err)
	}
	logger.Info("All detectors cleaned up")

	// TODO: Cleanup UI

	logger.Info("Application stopped")
}
