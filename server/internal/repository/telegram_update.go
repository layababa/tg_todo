package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// TelegramUpdate represents a raw update from Telegram
type TelegramUpdate struct {
	ID        int64          `gorm:"primaryKey"`
	UpdateID  int64          `gorm:"uniqueIndex;not null"`
	RawData   datatypes.JSON `gorm:"type:jsonb;not null"`
	Processed bool           `gorm:"default:false"`
	CreatedAt time.Time
}

// TelegramUpdateRepository handles database operations for telegram updates
type TelegramUpdateRepository interface {
	Save(ctx context.Context, update *TelegramUpdate) error
	GetRecentMessages(ctx context.Context, chatID int64, limit int) ([]TelegramUpdate, error)
}

type telegramUpdateRepository struct {
	db *gorm.DB
}

// NewTelegramUpdateRepository creates a new instance of TelegramUpdateRepository
func NewTelegramUpdateRepository(db *gorm.DB) TelegramUpdateRepository {
	return &telegramUpdateRepository{db: db}
}

// Save persists a telegram update to the database
func (r *telegramUpdateRepository) Save(ctx context.Context, update *TelegramUpdate) error {
	return r.db.WithContext(ctx).Create(update).Error
}

// GetRecentMessages retrieves the last N messages for a chat
func (r *telegramUpdateRepository) GetRecentMessages(ctx context.Context, chatID int64, limit int) ([]TelegramUpdate, error) {
	var updates []TelegramUpdate
	// Query JSONB: raw_data->'message'->'chat'->'id'
	// Note: ->> returns text, so we must compare with string representation of chatID
	chatIDStr := fmt.Sprintf("%d", chatID)
	err := r.db.WithContext(ctx).
		Where("raw_data -> 'message' -> 'chat' ->> 'id' = ?", chatIDStr).
		Order("created_at DESC").
		Limit(limit).
		Find(&updates).Error
	if err != nil {
		return nil, err
	}
	// Reverse order to be chronological? Usually nice for context.
	// But slice reverse is manual in Go.
	// We can return DESC and let service reverse or usage decide.
	// Creator usage implies chronological context list.
	// Let's keep it DESC here (newest first) for efficient DB query, then caller might reverse.
	// Actually for "Context Snapshot", chronological (oldest to newest) makes sense for reading.
	// But to get "last 10", we MUST sort DESC Limit 10, THEN reverse.
	return updates, nil
}
