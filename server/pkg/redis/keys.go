package redis

import "fmt"

const (
	// TelegramUpdateKey is the key for storing/locking telegram updates
	// Format: telegram:update:{update_id}
	TelegramUpdateKey = "telegram:update:%d"
)

// GetTelegramUpdateKey returns the redis key for a given update ID
func GetTelegramUpdateKey(updateID int64) string {
	return fmt.Sprintf(TelegramUpdateKey, updateID)
}
