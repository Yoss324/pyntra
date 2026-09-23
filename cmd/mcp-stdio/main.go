package main

import (
	"pyntra/internal/config"
	"pyntra/internal/logger"
	"pyntra/internal/mcp"
	"pyntra/internal/security"
	"flag"
	"fmt"
	"os"

	"go.uber.org/zap"
)

func main() {
	var configPath = flag.String("config", "config.yaml", "Configuration file path")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger (stdio mode uses stderr for logging to avoid interfering with JSON-RPC communication)
	log := logger.New(cfg.Log.Level, "stderr")

	// Create MCP server
	mcpServer := mcp.NewServer(log.Logger)

	// Create security tool executor
	executor := security.NewExecutor(&cfg.Security, mcpServer, log.Logger)

	// Register tools
	executor.RegisterTools(mcpServer)

	log.Logger.Info("MCP server (stdio mode) started, waiting for messages...")

	// Run stdio loop
	if err := mcpServer.HandleStdio(); err != nil {
		log.Logger.Error("MCP server execution failed", zap.Error(err))
		os.Exit(1)
	}
}

