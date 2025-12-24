package repository

import (
	"context"

	"github.com/layababa/tg_todo/server/internal/models"
	"gorm.io/gorm"
)

// UserGroupRepository defines the interface for user-group operations
type UserGroupRepository interface {
	FindByUserAndGroup(ctx context.Context, userID, groupID string) (*models.UserGroup, error)
	Create(ctx context.Context, userGroup *models.UserGroup) error
	Delete(ctx context.Context, userID, groupID string) error
}

type userGroupRepository struct {
	db *gorm.DB
}

// NewUserGroupRepository creates a new UserGroupRepository
func NewUserGroupRepository(db *gorm.DB) UserGroupRepository {
	return &userGroupRepository{db: db}
}

// FindByUserAndGroup finds a user-group relationship
func (r *userGroupRepository) FindByUserAndGroup(ctx context.Context, userID, groupID string) (*models.UserGroup, error) {
	var userGroup models.UserGroup
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND group_id = ?", userID, groupID).
		First(&userGroup).Error
	if err != nil {
		return nil, err
	}
	return &userGroup, nil
}

// Create creates a new user-group relationship
func (r *userGroupRepository) Create(ctx context.Context, userGroup *models.UserGroup) error {
	return r.db.WithContext(ctx).Create(userGroup).Error
}

// Delete removes a user-group relationship
func (r *userGroupRepository) Delete(ctx context.Context, userID, groupID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND group_id = ?", userID, groupID).
		Delete(&models.UserGroup{}).Error
}
