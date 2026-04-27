package repo

import (
	"context"

	"github.com/LurusTech/lurus-hub/internal/domain/entity"
)

// AuditEventRepo implements governance.AuditWriter.
type AuditEventRepo struct{}

// CreateAuditEvent persists an audit event to the database.
func (r *AuditEventRepo) CreateAuditEvent(event *entity.AuditEvent) error {
	return DB.Create(event).Error
}

// GetAuditEvents queries audit events with filters and pagination.
func GetAuditEvents(tenantID string, action string, actorID int, resource string, startTime, endTime int64, offset, limit int) (events []*entity.AuditEvent, total int64, err error) {
	tx := DB.Model(&entity.AuditEvent{})

	if tenantID != "" {
		tx = tx.Where("tenant_id = ?", tenantID)
	}
	if action != "" {
		tx = tx.Where("action = ?", action)
	}
	if actorID > 0 {
		tx = tx.Where("actor_id = ?", actorID)
	}
	if resource != "" {
		tx = tx.Where("resource = ?", resource)
	}
	if startTime > 0 {
		tx = tx.Where("timestamp >= ?", startTime)
	}
	if endTime > 0 {
		tx = tx.Where("timestamp <= ?", endTime)
	}

	if err = tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	err = tx.Order("timestamp DESC").Offset(offset).Limit(limit).Find(&events).Error
	return events, total, err
}

// DeleteOldAuditEvents deletes audit events older than targetTimestamp in batches.
// Mirrors DeleteOldLog pattern for consistent retention management.
func DeleteOldAuditEvents(ctx context.Context, targetTimestamp int64, limit int) (int64, error) {
	var total int64
	for {
		if ctx.Err() != nil {
			return total, ctx.Err()
		}
		result := DB.Where("timestamp < ?", targetTimestamp).Limit(limit).Delete(&entity.AuditEvent{})
		if result.Error != nil {
			return total, result.Error
		}
		total += result.RowsAffected
		if result.RowsAffected < int64(limit) {
			break
		}
	}
	return total, nil
}
