package task

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
)

// RepositoryPort 描述 Service 依赖的最小接口。
type RepositoryPort interface {
	List(ctx context.Context) ([]*Task, error)
	Get(ctx context.Context, id string) (*Task, error)
	Create(ctx context.Context, input CreateInput) (*Task, error)
	Update(ctx context.Context, id string, title *string, status *Status) (*Task, error)
	Delete(ctx context.Context, id string) error
}

// ErrInvalidInput 表示创建/更新任务时的参数校验错误。
var ErrInvalidInput = errors.New("invalid task payload")

// Service 处理与任务相关的业务逻辑。
type Service struct {
	repo     RepositoryPort
	notifier Notifier
}

// NewService 构造任务服务。
func NewService(repo RepositoryPort, notifier Notifier) *Service {
	return &Service{repo: repo, notifier: notifier}
}

// List 返回任务集合。
func (s *Service) List(ctx context.Context) ([]*Task, error) {
	return s.repo.List(ctx)
}

// Get 返回单个任务。
func (s *Service) Get(ctx context.Context, id string) (*Task, error) {
	return s.repo.Get(ctx, id)
}

// Create 新增任务并返回创建结果。
func (s *Service) Create(ctx context.Context, input CreateInput) (*Task, error) {
	if strings.TrimSpace(input.Title) == "" {
		return nil, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if strings.TrimSpace(input.Creator.ID) == "" || strings.TrimSpace(input.Creator.DisplayName) == "" {
		return nil, fmt.Errorf("%w: creator is required", ErrInvalidInput)
	}
	if input.Status == "" {
		input.Status = StatusPending
	} else if input.Status != StatusPending && input.Status != StatusCompleted {
		return nil, fmt.Errorf("%w: invalid status %q", ErrInvalidInput, input.Status)
	}

	validAssignees := make([]Person, 0, len(input.Assignees))
	for _, person := range input.Assignees {
		if strings.TrimSpace(person.ID) == "" || strings.TrimSpace(person.DisplayName) == "" {
			continue
		}
		validAssignees = append(validAssignees, person)
	}
	input.Assignees = validAssignees

	created, err := s.repo.Create(ctx, input)
	if err != nil {
		return nil, err
	}

	// 只在有指派人时才通知
	if s.notifier != nil && len(input.Assignees) > 0 {
		if err := s.notifier.TaskCreated(ctx, created, created.CreatedBy); err != nil {
			log.Printf("task: notify create failed: %v", err)
		}
	}
	return created, nil
}

// Update 修改任务。
func (s *Service) Update(ctx context.Context, id string, actor *Person, title *string, status *Status) (*Task, error) {
	notifyStatus := actor != nil && status != nil && s.notifier != nil
	var previous *Task
	var oldStatus Status
	var err error
	if notifyStatus {
		previous, err = s.repo.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		oldStatus = previous.Status
	}

	updated, err := s.repo.Update(ctx, id, title, status)
	if err != nil {
		return nil, err
	}

	if notifyStatus && previous != nil && oldStatus != updated.Status {
		if err := s.notifier.TaskStatusChanged(ctx, updated, *actor, oldStatus, updated.Status); err != nil {
			log.Printf("task: notify status change failed: %v", err)
		}
	}
	return updated, nil
}

// Delete 移除任务。
func (s *Service) Delete(ctx context.Context, id string, actor *Person) error {
	notifyDelete := actor != nil && s.notifier != nil
	var target *Task
	var err error
	if notifyDelete {
		target, err = s.repo.Get(ctx, id)
		if err != nil {
			return err
		}
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	if notifyDelete && target != nil {
		if err := s.notifier.TaskDeleted(ctx, target, *actor); err != nil {
			log.Printf("task: notify delete failed: %v", err)
		}
	}
	return nil
}
