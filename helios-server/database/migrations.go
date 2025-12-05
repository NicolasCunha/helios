// Package database provides schema migrations for the Helios database.
package database

import (
	"log"
)

// migrate runs all database migrations to create the schema.
// Creates tables for health check logs, action logs, and event logs.
//
// Returns an error if any migration fails.
func migrate() error {
	migrations := []struct {
		name string
		sql  string
	}{
		{
			name: "create_health_check_logs_table",
			sql: `
CREATE TABLE IF NOT EXISTS health_check_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    container_id TEXT NOT NULL,
    container_name TEXT NOT NULL,
    status TEXT NOT NULL,
    resource_cpu REAL,
    resource_memory INTEGER,
    resource_memory_limit INTEGER,
    resource_network_rx INTEGER,
    resource_network_tx INTEGER,
    error_message TEXT,
    checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (container_id) REFERENCES containers(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_health_logs_container ON health_check_logs(container_id);
CREATE INDEX IF NOT EXISTS idx_health_logs_checked_at ON health_check_logs(checked_at);
CREATE INDEX IF NOT EXISTS idx_health_logs_status ON health_check_logs(status);
			`,
		},
		{
			name: "create_action_logs_table",
			sql: `
CREATE TABLE IF NOT EXISTS action_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    action_type TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    resource_name TEXT,
    success BOOLEAN NOT NULL DEFAULT 0,
    error_message TEXT,
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_action_logs_resource ON action_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_action_logs_executed_at ON action_logs(executed_at);
CREATE INDEX IF NOT EXISTS idx_action_logs_action_type ON action_logs(action_type);
			`,
		},
		{
			name: "create_event_logs_table",
			sql: `
CREATE TABLE IF NOT EXISTS event_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,
    level TEXT NOT NULL,
    message TEXT NOT NULL,
    metadata TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_event_logs_type ON event_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_event_logs_level ON event_logs(level);
CREATE INDEX IF NOT EXISTS idx_event_logs_created_at ON event_logs(created_at);
			`,
		},
	}

	for _, migration := range migrations {
		log.Printf("Running migration: %s", migration.name)
		if _, err := db.Exec(migration.sql); err != nil {
			log.Printf("Migration failed for %s: %v", migration.name, err)
			return err
		}
		log.Printf("Migration completed: %s", migration.name)
	}

	return nil
}
