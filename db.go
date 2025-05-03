package main

import (
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/tursodatabase/go-libsql"
)

func initMigration(db *sql.DB) {
	// create table if not exists
	slog.Info("Creating table articles")
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS articles (
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			sources TEXT NOT NULL,
			category TEXT NOT NULL,
			ai_model TEXT NOT NULL,
			created_at TEXT NOT NULL
		)
	`)
	if err != nil {
		slog.Error("Error creating table articles", "error", err)
		os.Exit(1)
	}
}

func InitDB() (*sql.DB, func(), error) {
	dbName := "local.db"
	primaryUrl := os.Getenv("TURSO_DATABASE_URL")
	authToken := os.Getenv("TURSO_AUTH_TOKEN")

	dbPath := filepath.Join("./", dbName)
	slog.Info("Using database path", "path", dbPath)

	// Create connector with sync interval
	connector, err := libsql.NewEmbeddedReplicaConnector(dbPath, primaryUrl,
		libsql.WithAuthToken(authToken),
		libsql.WithSyncInterval(time.Minute),
	)
	if err != nil {
		return nil, nil, err
	}

	// Open database with the connector
	db := sql.OpenDB(connector)

	// Initialize migration
	initMigration(db)

	// Create cleanup function to be called later
	cleanup := func() {
		slog.Info("Cleaning up database resources")
		if err := db.Close(); err != nil {
			slog.Error("Error closing database", "error", err)
		}
	}

	return db, cleanup, nil
}
