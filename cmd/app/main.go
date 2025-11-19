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

	// 初始化日志记录器
	if _, err := logger.Setup(logger.INFO); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Logger initialized")
	logger.Infof("Application: %s", version.GetFullName())
	logger.Infof("Version: %s", version.Version)
	logger.Infof("Author: %s", version.Author)

	// 加载配置
	cfg, err := config.Get()
	if err != nil {
		logger.Errorf("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	logger.Info("Configuration loaded successfully")
	logger.Debugf("Update interval: %.2f seconds", cfg.UpdateInterval)
	logger.Debugf("Time scale: %.2f", cfg.TimeScale)

	// 初始化检测器注册表
	registry := detector.NewDetectorRegistry()

	// 创建并注册检测器
	dayDetector := detector.NewDayDetector(cfg)
	registry.Register(dayDetector)

	// 初始化所有检测器
	logger.Info("Initializing detectors...")
	if err := registry.InitializeAll(); err != nil {
		logger.Errorf("Failed to initialize detectors: %v", err)
		os.Exit(1)
	}
	logger.Info("All detectors initialized successfully")

	// 创建更新器
	upd := updater.NewUpdater(cfg, registry)

	// TODO: 初始化 UI
	logger.Info("TODO: Initialize UI")

	// 创建用于优雅关闭的上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动更新器
	logger.Info("Starting updater...")
	if err := upd.Start(ctx); err != nil {
		logger.Errorf("Failed to start updater: %v", err)
		os.Exit(1)
	}
	logger.Info("Updater started successfully")

	logger.Info("Application started successfully")
	logger.Info("Press Ctrl+C to exit")

	// 等待中断信号以优雅地关闭
	quit := make(chan os.Signal, 1)
	// 接受 SIGINT (Ctrl+C) 和 SIGTERM
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞直到收到信号
	sig := <-quit
	logger.Infof("Received signal: %v", sig)
	logger.Info("Shutting down gracefully...")

	// 取消上下文
	cancel()

	// 停止更新器
	if err := upd.Stop(); err != nil {
		logger.Errorf("Error stopping updater: %v", err)
	}

	// 清理检测器
	logger.Info("Cleaning up detectors...")
	if err := registry.CleanupAll(); err != nil {
		logger.Errorf("Error cleaning up detectors: %v", err)
	}
	logger.Info("All detectors cleaned up")

	// TODO: 清理 UI

	logger.Info("Application stopped")
}
