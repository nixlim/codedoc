package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configure logging
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	
	// Set log level from environment
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		log.Fatal().Err(err).Msg("Invalid log level")
	}
	zerolog.SetGlobalLevel(level)
	
	// Log startup
	log.Info().
		Str("version", "0.0.1").
		Str("log_level", logLevel).
		Msg("Starting CodeDoc MCP Server")
	
	// Placeholder for server initialization
	fmt.Println("CodeDoc MCP Server - Foundation Ready")
}