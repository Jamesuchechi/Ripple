package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"ripple/internal/config"
)

func main() {
	direction := flag.String("dir", "up", "Migration direction: up or down")
	flag.Parse()

	if len(os.Args) > 1 && (os.Args[1] == "up" || os.Args[1] == "down") {
		*direction = os.Args[1]
	}

	cfg := config.Load()
	migrationsPath := "file://migrations"

	m, err := migrate.New(migrationsPath, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize migration engine: %v", err)
	}
	defer m.Close()

	switch *direction {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration up failed: %v", err)
		}
		fmt.Println("Database migration UP completed successfully.")
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration down failed: %v", err)
		}
		fmt.Println("Database migration DOWN completed successfully.")
	default:
		log.Fatalf("Unknown migration direction: %s. Use 'up' or 'down'.", *direction)
	}
}
