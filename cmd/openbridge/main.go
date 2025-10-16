package main

import (
	"flag"
	"fmt"
	"log"
	"GopenBridge/config"
	"GopenBridge/server"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Parse CLI flags
	host := flag.String("host", cfg.Host, "Host to bind to")
	port := flag.Int("port", cfg.Port, "Port to bind to")
	reload := flag.Bool("reload", false, "Enable auto-reload for development (not supported)")
	flag.Parse()

	// Print configuration info
	config.PrintConfigInfo(cfg)
	fmt.Println()

	// Start server
	fmt.Printf("🌉 GopenBridge proxy starting on %s:%d\n", *host, *port)
	fmt.Printf("📋 Config: ANTHROPIC_BASE_URL=http://%s:%d/\n", *host, *port)
	// Update config host and port
	cfg.Host = *host
	cfg.Port = *port
	_ = reload // reload flag not implemented
	if err := server.StartServer(cfg); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
