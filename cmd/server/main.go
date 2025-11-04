package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ZindGH/MQTT-Server/internal/server"
)

func main() {
	log.Println("Starting MQTT Server...")

	// Create and start the server
	srv, err := server.New()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	log.Println("MQTT Server started successfully on port 1883")

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := srv.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
	fmt.Println("Server stopped gracefully")
}
