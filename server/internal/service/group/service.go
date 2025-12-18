package group

import (
	"context"
	"errors"
	"fmt"

	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/internal/repository"
	notionsvc "github.com/layababa/tg_todo/server/internal/service/notion"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrGroupNotFound  = errors.New("group not found")
	ErrNotAdmin       = errors.New("user is not an admin of this group")
	ErrNotionNotBound = errors.New("notion not fully configured for user")
)

type GroupSummary struct {
	ID     string                     `json:"id"`
	Title  string                     `json:"title"`
	Status models.GroupStatus         `json:"status"`
	DB     *notionsvc.DatabaseSummary `json:"db"`
	Role   models.GroupRole           `json:"role"`
}

type Service struct {
	logger        *zap.Logger
	groupRepo     repository.GroupRepository
	notionService *notionsvc.Service
}

func NewService(logger *zap.Logger, groupRepo repository.GroupRepository, notionService *notionsvc.Service) *Service {
	return &Service{
		logger:        logger,
		groupRepo:     groupRepo,
		notionService: notionService,
	}
}

func (s *Service) ListGroups(ctx context.Context, userID string) ([]GroupSummary, error) {
	groups, err := s.groupRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch groups: %w", err)
	}

	summaries := make([]GroupSummary, 0, len(groups))
	for _, g := range groups {
		// Check role
		_, rolePtr, _ := s.groupRepo.IsMember(ctx, userID, g.ID)
		role := models.GroupRoleMember
		if rolePtr != nil {
			role = *rolePtr
		}

		var dbSummary *notionsvc.DatabaseSummary
		if g.DatabaseID != nil {
			dbSummary = &notionsvc.DatabaseSummary{
				ID:   *g.DatabaseID,
				Name: g.DatabaseName,
				// Icon and Workspace handled by frontend or need Notion fetch?
				// For now simple struct.
			}
		}

		summaries = append(summaries, GroupSummary{
			ID:     g.ID,
			Title:  g.Title,
			Status: g.Status,
			DB:     dbSummary,
			Role:   role,
		})
	}

	return summaries, nil
}

func (s *Service) ValidateDatabase(ctx context.Context, userID, groupID, dbID string) (*notionsvc.ValidationResult, error) {
	// Check if user is admin
	isAdmin, err := s.checkAdmin(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, ErrNotAdmin
	}

	return s.notionService.ValidateDatabase(ctx, userID, dbID)
}

func (s *Service) InitDatabase(ctx context.Context, userID, groupID, dbID string) (*notionsvc.InitResult, error) {
	// Check if user is admin
	isAdmin, err := s.checkAdmin(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, ErrNotAdmin
	}

	return s.notionService.InitializeDatabase(ctx, userID, dbID)
}

func (s *Service) BindDatabase(ctx context.Context, userID, groupID, dbID string) (*models.Group, error) {
	// Check if user is admin
	isAdmin, err := s.checkAdmin(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, ErrNotAdmin
	}

	// Fetch DB info to get name (Validation also fetches it, maybe optimization later)
	validation, err := s.notionService.ValidateDatabase(ctx, userID, dbID)
	if err != nil {
		return nil, err
	}
	// We allow binding even if not fully compatible?
	// Usually strict binding is better, but MVP: let's assume if user explicitly binds, they might fix it later or used init.
	// But `Bind` implies "Connect".

	group, err := s.groupRepo.FindByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrGroupNotFound
	}

	group.DatabaseID = &dbID
	group.DatabaseName = validation.Name
	group.Status = models.GroupStatusConnected

	err = s.groupRepo.CreateOrUpdate(ctx, group)
	if err != nil {
		return nil, err
	}

	return group, nil
}

func (s *Service) UnbindDatabase(ctx context.Context, userID, groupID string) (*models.Group, error) {
	// Check if user is admin
	isAdmin, err := s.checkAdmin(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, ErrNotAdmin
	}

	group, err := s.groupRepo.FindByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrGroupNotFound
	}

	// Unbind
	group.DatabaseID = nil
	group.DatabaseName = ""
	group.Status = models.GroupStatusUnbound

	err = s.groupRepo.CreateOrUpdate(ctx, group)
	if err != nil {
		return nil, err
	}

	return group, nil
}

func (s *Service) checkAdmin(ctx context.Context, userID, groupID string) (bool, error) {
	isMember, role, err := s.groupRepo.IsMember(ctx, userID, groupID)
	if err != nil {
		return false, err
	}
	if !isMember || role == nil {
		return false, nil
	}
	return *role == models.GroupRoleAdmin, nil
}

// EnsureGroup ensures a group exists and the user is an admin.
// Called when bot is added to a group or interacts with an admin in a group.
func (s *Service) EnsureGroup(ctx context.Context, groupID, title, addedByUserID string) error {
	group, err := s.groupRepo.FindByID(ctx, groupID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if group == nil {
		group = &models.Group{
			ID:     groupID,
			Title:  title,
			Status: models.GroupStatusUnbound,
		}
	} else {
		// Update title if changed
		group.Title = title
		if group.Status == models.GroupStatusInactive {
			group.Status = models.GroupStatusUnbound
		}
	}

	if err := s.groupRepo.CreateOrUpdate(ctx, group); err != nil {
		return err
	}

	// add admin
	if addedByUserID != "" {
		if err := s.groupRepo.AddMember(ctx, addedByUserID, groupID, models.GroupRoleAdmin); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) UpdateStatus(ctx context.Context, groupID string, status models.GroupStatus) error {
	group, err := s.groupRepo.FindByID(ctx, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return ErrGroupNotFound
	}

	group.Status = status
	return s.groupRepo.CreateOrUpdate(ctx, group)
}
