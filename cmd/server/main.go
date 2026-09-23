package main

import (
	"pyntra/internal/app"
	"pyntra/internal/config"
	"pyntra/internal/logger"
	"flag"
	"fmt"
)

func main() {
	var configPath = flag.String("config", "config.yaml", "Configuration file path")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		return
	}

	// When MCP is enabled and auth_header_value is empty, automatically generate a random key and write it back to the configuration
	if err := config.EnsureMCPAuth(*configPath, cfg); err != nil {
		fmt.Printf("MCP authentication configuration failed: %v\n", err)
		return
	}
	if cfg.MCP.Enabled {
		config.PrintMCPConfigJSON(cfg.MCP)
		config.PrintMCPClientConfigs(cfg.MCP)
	}

	// Initialize logger
	log := logger.New(cfg.Log.Level, cfg.Log.Output)

	// Create application
	application, err := app.New(cfg, log)
	if err != nil {
		log.Fatal("Application initialization failed", "error", err)
	}

	// Start server
	if err := application.Run(); err != nil {
		log.Fatal("Server startup failed", "error", err)
	}
}

