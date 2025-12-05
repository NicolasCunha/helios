// Package repository provides data access layer for logs.
package repository

import (
	"database/sql"

	"nfcunha/helios/core/models"
)

// EventLogRepository handles persistence of event logs.
type EventLogRepository struct {
	db *sql.DB
}

// NewEventLogRepository creates a new event log repository.
func NewEventLogRepository(db *sql.DB) *EventLogRepository {
	return &EventLogRepository{db: db}
}

// Create stores an event log in the database.
func (r *EventLogRepository) Create(log *models.EventLog) error {
	query := `
		INSERT INTO event_logs (event_type, level, message, metadata, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	var metadata *string
	if log.Metadata != "" {
		metadata = &log.Metadata
	}

	result, err := r.db.Exec(
		query,
		log.EventType,
		log.Level,
		log.Message,
		metadata,
		log.CreatedAt,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	log.ID = id

	return nil
}

// GetRecent retrieves recent event logs.
func (r *EventLogRepository) GetRecent(limit int) ([]*models.EventLog, error) {
	query := `
		SELECT id, event_type, level, message, metadata, created_at
		FROM event_logs
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.EventLog
	for rows.Next() {
		log := &models.EventLog{}
		var metadata sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.EventType,
			&log.Level,
			&log.Message,
			&metadata,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if metadata.Valid {
			log.Metadata = metadata.String
		}

		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// GetByType retrieves event logs filtered by type.
func (r *EventLogRepository) GetByType(eventType string, limit int) ([]*models.EventLog, error) {
	query := `
		SELECT id, event_type, level, message, metadata, created_at
		FROM event_logs
		WHERE event_type = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, eventType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.EventLog
	for rows.Next() {
		log := &models.EventLog{}
		var metadata sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.EventType,
			&log.Level,
			&log.Message,
			&metadata,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if metadata.Valid {
			log.Metadata = metadata.String
		}

		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// DeleteOlderThan removes event logs older than the specified duration.
func (r *EventLogRepository) DeleteOlderThan(days int) (int64, error) {
	query := `DELETE FROM event_logs WHERE created_at < datetime('now', '-' || ? || ' days')`
	result, err := r.db.Exec(query, days)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
