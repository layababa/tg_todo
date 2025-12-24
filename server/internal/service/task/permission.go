package task

import (
	"context"

	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/internal/repository"
)

// UserGroupRepository defines the interface for user-group operations
type UserGroupRepository interface {
	FindByUserAndGroup(ctx context.Context, userID, groupID string) (*models.UserGroup, error)
}

// CanModifyTask checks if a user has permission to modify a task
// Returns true if user is: group admin, task creator, or task assignee
func CanModifyTask(ctx context.Context, userID string, task *repository.Task, userGroupRepo UserGroupRepository) (bool, error) {
	// 1. Check if user is task creator
	if task.CreatorID != nil && *task.CreatorID == userID {
		return true, nil
	}

	// 2. Check if user is assignee
	for _, assignee := range task.Assignees {
		if assignee.ID == userID {
			return true, nil
		}
	}

	// 3. Check if user is group admin
	if task.GroupID != nil {
		userGroup, err := userGroupRepo.FindByUserAndGroup(ctx, userID, *task.GroupID)
		if err == nil && userGroup.Role == models.GroupRoleAdmin {
			return true, nil
		}
		// If error is "not found", user is not in group or not admin, continue to return false
		// Other errors are ignored for now (fail-open approach)
	}

	return false, nil
}
