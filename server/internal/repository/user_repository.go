package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/layababa/tg_todo/server/internal/models"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	FindByTgID(ctx context.Context, tgID int64) (*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	FindNotionToken(ctx context.Context, userID string) (*models.UserNotionToken, error)
	SaveNotionToken(ctx context.Context, token *models.UserNotionToken) error
	ListAll(ctx context.Context) ([]models.User, error)
}

// userRepository implements UserRepository using GORM
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new instance of UserRepository
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// FindByTgID finds a user by their Telegram ID
func (r *userRepository) FindByTgID(ctx context.Context, tgID int64) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("tg_id = ?", tgID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID finds a user by UUID
func (r *userRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Create creates a new user
func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// Update updates an existing user
func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// GetByUsername finds a user by their Telegram username
func (r *userRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("tg_username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindNotionToken finds the Notion token for a user
func (r *userRepository) FindNotionToken(ctx context.Context, userID string) (*models.UserNotionToken, error) {
	var token models.UserNotionToken
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// SaveNotionToken saves or updates a Notion token
func (r *userRepository) SaveNotionToken(ctx context.Context, token *models.UserNotionToken) error {
	// Use Upsert pattern: try to update, if not found, create
	// GORM's Save creates if not exists or updates primary key
	return r.db.WithContext(ctx).Save(token).Error
}

// ListAll returns all users
func (r *userRepository) ListAll(ctx context.Context) ([]models.User, error) {
	var users []models.User
	if err := r.db.WithContext(ctx).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
