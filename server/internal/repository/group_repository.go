package repository

import (
	"context"

	"github.com/layababa/tg_todo/server/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GroupRepository interface {
	FindByID(ctx context.Context, id string) (*models.Group, error)
	FindByUserID(ctx context.Context, userID string) ([]models.Group, error)
	CreateOrUpdate(ctx context.Context, group *models.Group) error
	AddMember(ctx context.Context, userID, groupID string, role models.GroupRole) error
	IsMember(ctx context.Context, userID, groupID string) (bool, *models.GroupRole, error)
	ListWithActiveBindings(ctx context.Context) ([]models.Group, error)
}

type groupRepository struct {
	db *gorm.DB
}

func NewGroupRepository(db *gorm.DB) GroupRepository {
	return &groupRepository{db: db}
}

func (r *groupRepository) FindByID(ctx context.Context, id string) (*models.Group, error) {
	var group models.Group
	err := r.db.WithContext(ctx).First(&group, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *groupRepository) FindByUserID(ctx context.Context, userID string) ([]models.Group, error) {
	var groups []models.Group
	// Join with user_groups table
	err := r.db.WithContext(ctx).
		Table("groups").
		Joins("JOIN user_groups ON user_groups.group_id = groups.id").
		Where("user_groups.user_id = ?", userID).
		Find(&groups).Error
	return groups, err
}

func (r *groupRepository) CreateOrUpdate(ctx context.Context, group *models.Group) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(group).Error
}

func (r *groupRepository) AddMember(ctx context.Context, userID, groupID string, role models.GroupRole) error {
	member := models.UserGroup{
		UserID:  userID,
		GroupID: groupID,
		Role:    role,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&member).Error
}

func (r *groupRepository) IsMember(ctx context.Context, userID, groupID string) (bool, *models.GroupRole, error) {
	var member models.UserGroup
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND group_id = ?", userID, groupID).
		First(&member).Error

	if err == gorm.ErrRecordNotFound {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	return true, &member.Role, nil
}

func (r *groupRepository) ListWithActiveBindings(ctx context.Context) ([]models.Group, error) {
	var groups []models.Group
	err := r.db.WithContext(ctx).
		Where("database_id IS NOT NULL AND notion_access_token != ''").
		Find(&groups).Error
	return groups, err
}
