package task

import "context"

// RepositoryPort 描述 Service 依赖的最小接口。
type RepositoryPort interface {
	List(ctx context.Context) ([]*Task, error)
	Get(ctx context.Context, id string) (*Task, error)
	Update(ctx context.Context, id string, title *string, status *Status) (*Task, error)
	Delete(ctx context.Context, id string) error
}

// Service 处理与任务相关的业务逻辑。
type Service struct {
	repo RepositoryPort
}

// NewService 构造任务服务。
func NewService(repo RepositoryPort) *Service {
	return &Service{repo: repo}
}

// List 返回任务集合。
func (s *Service) List(ctx context.Context) ([]*Task, error) {
	return s.repo.List(ctx)
}

// Get 返回单个任务。
func (s *Service) Get(ctx context.Context, id string) (*Task, error) {
	return s.repo.Get(ctx, id)
}

// Update 修改任务。
func (s *Service) Update(ctx context.Context, id string, title *string, status *Status) (*Task, error) {
	return s.repo.Update(ctx, id, title, status)
}

// Delete 移除任务。
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
