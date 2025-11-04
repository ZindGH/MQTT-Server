package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/ZindGH/MQTT-Server/internal/config"
	"github.com/ZindGH/MQTT-Server/internal/server"
	"github.com/ZindGH/MQTT-Server/internal/store"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config/config.yaml", "Path to configuration file")
	flag.Parse()

	log.Println("Starting MQTT Server...")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded from %s", *configPath)
	log.Printf("Server will bind to %s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Storage backend: %s", cfg.Storage.Backend)
	log.Printf("Max QoS level: %d", cfg.QoS.MaxQoS)

	// Initialize storage
	var st store.Store
	switch cfg.Storage.Backend {
	case "bbolt":
		// Ensure data directory exists
		dir := filepath.Dir(cfg.Storage.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create data directory: %v", err)
		}

		st, err = store.NewBboltStore(cfg.Storage.Path)
		if err != nil {
			log.Fatalf("Failed to initialize bbolt store: %v", err)
		}
		log.Printf("Bbolt storage initialized at %s", cfg.Storage.Path)
		defer st.Close()

	case "memory":
		log.Println("Using in-memory storage (data will not persist)")
		// TODO: Implement memory store
		log.Fatal("Memory storage not yet implemented")

	default:
		log.Fatalf("Unsupported storage backend: %s", cfg.Storage.Backend)
	}

	// Create server with configuration and storage
	srv, err := server.NewWithConfig(cfg, st)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start Prometheus metrics server if enabled
	if cfg.Metrics.Enabled {
		go func() {
			metricsAddr := fmt.Sprintf(":%d", cfg.Metrics.Port)
			http.Handle(cfg.Metrics.Path, promhttp.Handler())
			log.Printf("Metrics server starting on %s%s", metricsAddr, cfg.Metrics.Path)
			if err := http.ListenAndServe(metricsAddr, nil); err != nil {
				log.Printf("Metrics server error: %v", err)
			}
		}()
	}

	// Start MQTT server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("Server stopped: %v", err)
		}
	}()

	log.Println("✓ MQTT Server started successfully")
	log.Printf("  → MQTT listening on %s:%d", cfg.Server.Host, cfg.Server.Port)
	if cfg.Metrics.Enabled {
		log.Printf("  → Metrics available at http://localhost:%d%s", cfg.Metrics.Port, cfg.Metrics.Path)
	}
	log.Printf("  → Log level: %s", cfg.Logging.Level)
	log.Println("Press Ctrl+C to stop")

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("\nShutting down server...")
	if err := srv.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
	fmt.Println("✓ Server stopped gracefully")
}
