// Package database provides database initialization and connection management.
package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// Initialize opens a connection to the SQLite database and runs migrations.
// The database path is provided as a parameter from the configuration.
// Returns an error if the database cannot be opened or migrations fail.
func Initialize(dbPath string) error {
	log.Printf("Initializing database at: %s", dbPath)

	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Printf("Failed to open database: %v", err)
		return err
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	// Test connection
	if err := db.Ping(); err != nil {
		log.Printf("Failed to ping database: %v", err)
		return err
	}

	// Run migrations
	if err := migrate(); err != nil {
		log.Printf("Failed to run migrations: %v", err)
		return err
	}

	log.Println("Database initialized successfully")
	return nil
}

// GetDB returns the active database connection.
// Initialize() must be called before using this function.
func GetDB() *sql.DB {
	return db
}

// Close closes the database connection.
// This should be called during application shutdown.
func Close() error {
	if db != nil {
		log.Println("Closing database connection")
		return db.Close()
	}
	return nil
}
