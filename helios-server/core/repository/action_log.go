// Package repository provides data access layer for logs.
package repository

import (
	"database/sql"

	"nfcunha/helios/core/models"
)

// ActionLogRepository handles persistence of action logs.
type ActionLogRepository struct {
	db *sql.DB
}

// NewActionLogRepository creates a new action log repository.
func NewActionLogRepository(db *sql.DB) *ActionLogRepository {
	return &ActionLogRepository{db: db}
}

// Create stores an action log in the database.
func (r *ActionLogRepository) Create(log *models.ActionLog) error {
	query := `
		INSERT INTO action_logs (
			action_type, resource_type, resource_id, resource_name,
			success, error_message, executed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	var errorMsg *string
	if log.ErrorMessage != "" {
		errorMsg = &log.ErrorMessage
	}

	var resourceName *string
	if log.ResourceName != "" {
		resourceName = &log.ResourceName
	}

	result, err := r.db.Exec(
		query,
		log.ActionType,
		log.ResourceType,
		log.ResourceID,
		resourceName,
		log.Success,
		errorMsg,
		log.ExecutedAt,
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

// GetByResource retrieves action logs for a specific resource.
func (r *ActionLogRepository) GetByResource(resourceType, resourceID string, limit int) ([]*models.ActionLog, error) {
	query := `
		SELECT id, action_type, resource_type, resource_id, resource_name,
		       success, error_message, executed_at
		FROM action_logs
		WHERE resource_type = ? AND resource_id = ?
		ORDER BY executed_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, resourceType, resourceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.ActionLog
	for rows.Next() {
		log := &models.ActionLog{}
		var errorMsg, resourceName sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.ActionType,
			&log.ResourceType,
			&log.ResourceID,
			&resourceName,
			&log.Success,
			&errorMsg,
			&log.ExecutedAt,
		)
		if err != nil {
			return nil, err
		}

		if errorMsg.Valid {
			log.ErrorMessage = errorMsg.String
		}
		if resourceName.Valid {
			log.ResourceName = resourceName.String
		}

		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// GetRecent retrieves recent action logs across all resources.
func (r *ActionLogRepository) GetRecent(limit int) ([]*models.ActionLog, error) {
	query := `
		SELECT id, action_type, resource_type, resource_id, resource_name,
		       success, error_message, executed_at
		FROM action_logs
		ORDER BY executed_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.ActionLog
	for rows.Next() {
		log := &models.ActionLog{}
		var errorMsg, resourceName sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.ActionType,
			&log.ResourceType,
			&log.ResourceID,
			&resourceName,
			&log.Success,
			&errorMsg,
			&log.ExecutedAt,
		)
		if err != nil {
			return nil, err
		}

		if errorMsg.Valid {
			log.ErrorMessage = errorMsg.String
		}
		if resourceName.Valid {
			log.ResourceName = resourceName.String
		}

		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// DeleteOlderThan removes action logs older than the specified duration.
func (r *ActionLogRepository) DeleteOlderThan(days int) (int64, error) {
	query := `DELETE FROM action_logs WHERE executed_at < datetime('now', '-' || ? || ' days')`
	result, err := r.db.Exec(query, days)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
