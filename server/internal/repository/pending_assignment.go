package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/layababa/tg_todo/server/internal/models"
)

// PendingAssignmentRepository defines the interface for managing pending assignments
type PendingAssignmentRepository interface {
	Create(ctx context.Context, assignment *models.PendingAssignment) error
	ListByUsername(ctx context.Context, username string) ([]models.PendingAssignment, error)
	Delete(ctx context.Context, id string) error
	DeleteByUsername(ctx context.Context, username string) error
}

type pendingAssignmentRepo struct {
	db *gorm.DB
}

// NewPendingAssignmentRepository creates a new repository instance
func NewPendingAssignmentRepository(db *gorm.DB) PendingAssignmentRepository {
	return &pendingAssignmentRepo{db: db}
}

func (r *pendingAssignmentRepo) Create(ctx context.Context, assignment *models.PendingAssignment) error {
	return r.db.WithContext(ctx).Create(assignment).Error
}

func (r *pendingAssignmentRepo) ListByUsername(ctx context.Context, username string) ([]models.PendingAssignment, error) {
	var results []models.PendingAssignment
	err := r.db.WithContext(ctx).Where("tg_username ILIKE ?", username).Find(&results).Error
	return results, err
}

func (r *pendingAssignmentRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.PendingAssignment{}, "id = ?", id).Error
}

func (r *pendingAssignmentRepo) DeleteByUsername(ctx context.Context, username string) error {
	return r.db.WithContext(ctx).Where("tg_username ILIKE ?", username).Delete(&models.PendingAssignment{}).Error
}
