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
	GetRecentMessages(ctx context.Context, chatID int64, limit int, beforeID int64) ([]TelegramUpdate, error)
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

// GetRecentMessages retrieves the last N messages for a chat, optionally before a specific update ID
func (r *telegramUpdateRepository) GetRecentMessages(ctx context.Context, chatID int64, limit int, beforeID int64) ([]TelegramUpdate, error) {
	var updates []TelegramUpdate
	chatIDStr := fmt.Sprintf("%d", chatID)

	query := r.db.WithContext(ctx).
		Where("raw_data -> 'message' -> 'chat' ->> 'id' = ?", chatIDStr)

	if beforeID > 0 {
		// assuming update_id is sequential, we filter by update_id < beforeID
		// Or should we use message_id?
		// TelegramUpdate stores raw JSON. We don't have indexed message_id column (only UpdateID).
		// However, we want context *before the referenced message*.
		// If we use UpdateID, we need to know the UpdateID of the referenced message.
		// We only have MessageID from ReplyToMessage.
		// Querying JSONB for message_id is slow but works for small datasets or with GIN index.
		// Ideally we should extract message_id to a column.
		// For now, let's query raw_data->'message'->'message_id' < ?
		query = query.Where("(raw_data -> 'message' ->> 'message_id')::int < ?", beforeID)
	}

	err := query.
		Order("created_at DESC").
		Limit(limit).
		Find(&updates).Error
	if err != nil {
		return nil, err
	}
	return updates, nil
}
