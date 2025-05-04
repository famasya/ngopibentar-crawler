package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	"github.com/tursodatabase/go-libsql"
)

func initMigration(db *sql.DB) {
	// create table if not exists
	logger.Debug("running migration 1")
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS articles (
			id BIGINT PRIMARY KEY NOT NULL,
			title TEXT NOT NULL,
			excerpt TEXT NOT NULL,
			long_content TEXT NOT NULL,
			sources TEXT NOT NULL, -- comma separated
			links TEXT NOT NULL, -- comma separated
			category TEXT NOT NULL,
			ai_model TEXT NOT NULL,
			publishers TEXT NOT NULL,
			created_at TEXT NOT NULL
		);

		CREATE UNIQUE INDEX IF NOT EXISTS idx_articles_id ON articles(id);
	`)
	if err != nil {
		logger.Error("Error creating table articles", "error", err)
		os.Exit(1)
	}

	logger.Debug("running migration 2")
	_, err = db.Exec(`
		ALTER TABLE articles DROP COLUMN publishers;
	`)
	if err != nil {
		logger.Debug("Error altering table articles", "error", err)
	}
}

func InitDB() (*sql.DB, func(), error) {
	dbName := "local.db"
	primaryUrl := os.Getenv("TURSO_DATABASE_URL")
	authToken := os.Getenv("TURSO_AUTH_TOKEN")

	dbPath := filepath.Join("./", dbName)
	logger.Info("Using database path", "path", dbPath)

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
		logger.Info("Cleaning up database resources")
		if err := db.Close(); err != nil {
			logger.Error("Error closing database", "error", err)
		}
	}

	return db, cleanup, nil
}
