package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ripple/internal/config"
)

func main() {
	cfg := config.Load()
	log.Printf("Starting Ripple Fanout Worker daemon (Concurrency limit: %d)...", cfg.FanoutConcurrency)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("Worker daemon polling for fanout queue messages...")
		case <-stop:
			log.Println("Shutting down worker daemon gracefully...")
			return
		}
	}
}
