// Package repository provides data access layer for logs.
package repository

import (
	"database/sql"

	"nfcunha/helios/core/models"
)

// HealthCheckLogRepository handles persistence of health check logs.
type HealthCheckLogRepository struct {
	db *sql.DB
}

// NewHealthCheckLogRepository creates a new health check log repository.
func NewHealthCheckLogRepository(db *sql.DB) *HealthCheckLogRepository {
	return &HealthCheckLogRepository{db: db}
}

// Create stores a health check result in the database.
func (r *HealthCheckLogRepository) Create(log *models.HealthCheckLog) error {
	query := `
		INSERT INTO health_check_logs (
			container_id, container_name, status,
			resource_cpu, resource_memory, resource_memory_limit,
			resource_network_rx, resource_network_tx,
			error_message, checked_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var errorMsg *string
	if log.ErrorMessage != "" {
		errorMsg = &log.ErrorMessage
	}

	result, err := r.db.Exec(
		query,
		log.ContainerID,
		log.ContainerName,
		log.Status,
		log.ResourceCPU,
		log.ResourceMemory,
		log.ResourceMemoryLimit,
		log.ResourceNetworkRx,
		log.ResourceNetworkTx,
		errorMsg,
		log.CheckedAt,
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

// GetByContainerID retrieves health check logs for a specific container.
func (r *HealthCheckLogRepository) GetByContainerID(containerID string, limit int) ([]*models.HealthCheckLog, error) {
	query := `
		SELECT id, container_id, container_name, status,
		       resource_cpu, resource_memory, resource_memory_limit,
		       resource_network_rx, resource_network_tx,
		       error_message, checked_at
		FROM health_check_logs
		WHERE container_id = ?
		ORDER BY checked_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, containerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.HealthCheckLog
	for rows.Next() {
		log := &models.HealthCheckLog{}
		var errorMsg sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.ContainerID,
			&log.ContainerName,
			&log.Status,
			&log.ResourceCPU,
			&log.ResourceMemory,
			&log.ResourceMemoryLimit,
			&log.ResourceNetworkRx,
			&log.ResourceNetworkTx,
			&errorMsg,
			&log.CheckedAt,
		)
		if err != nil {
			return nil, err
		}

		if errorMsg.Valid {
			log.ErrorMessage = errorMsg.String
		}

		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// DeleteOlderThan removes health check logs older than the specified duration.
func (r *HealthCheckLogRepository) DeleteOlderThan(days int) (int64, error) {
	query := `DELETE FROM health_check_logs WHERE checked_at < datetime('now', '-' || ? || ' days')`
	result, err := r.db.Exec(query, days)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
