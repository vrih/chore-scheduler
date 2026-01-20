package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps a sql.DB connection with additional functionality
type DB struct {
	*sql.DB
}

// New creates a new database connection
// If dbPath is ":memory:", an in-memory database is created
// Otherwise, the directory is created if it doesn't exist
func New(dbPath string) (*DB, error) {
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	sqlDB, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(1) // SQLite only supports one writer
	sqlDB.SetMaxIdleConns(1)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &DB{DB: sqlDB}, nil
}

// Initialize sets up the database schema
func (db *DB) Initialize() error {
	return runMigrations(db.DB)
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}
