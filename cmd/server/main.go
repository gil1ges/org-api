package main

import (
	"log"

	"org-api/internal/app"
	"org-api/internal/config"
	"org-api/internal/db"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	if err := db.RunMigrations(cfg.DSN(), "migrations"); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	database, err := db.ConnectPostgres(cfg.DSN())
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}

	server := app.NewServer(cfg, database)
	log.Printf("starting server on %s", cfg.Addr())
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
