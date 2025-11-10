package task

import (
	"context"
	"errors"
	"testing"
)

// MockRepository 模拟仓储层
type MockRepository struct {
	tasks map[string]*Task
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		tasks: make(map[string]*Task),
	}
}

func (m *MockRepository) List(ctx context.Context) ([]*Task, error) {
	tasks := make([]*Task, 0, len(m.tasks))
	for _, t := range m.tasks {
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (m *MockRepository) Get(ctx context.Context, id string) (*Task, error) {
	if t, ok := m.tasks[id]; ok {
		return t, nil
	}
	return nil, ErrNotFound
}

func (m *MockRepository) Create(ctx context.Context, input CreateInput) (*Task, error) {
	t := &Task{
		ID:        "123",
		Title:     input.Title,
		Status:    input.Status,
		CreatedBy: input.Creator,
		Assignees: input.Assignees,
	}
	m.tasks["123"] = t
	return t, nil
}

func (m *MockRepository) Update(ctx context.Context, id string, title *string, status *Status) (*Task, error) {
	t, err := m.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if title != nil {
		t.Title = *title
	}
	if status != nil {
		t.Status = *status
	}
	return t, nil
}

func (m *MockRepository) Delete(ctx context.Context, id string) error {
	if _, ok := m.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(m.tasks, id)
	return nil
}

// MockNotifier 模拟通知器
type MockNotifier struct {
	createdCalls       []MockNotifierCall
	statusChangedCalls []MockNotifierCall
	deletedCalls       []MockNotifierCall
}

type MockNotifierCall struct {
	TaskID   string
	Actor    Person
	OldValue string
	NewValue string
}

func (m *MockNotifier) TaskCreated(ctx context.Context, task *Task, creator Person) error {
	m.createdCalls = append(m.createdCalls, MockNotifierCall{
		TaskID: task.ID,
		Actor:  creator,
	})
	return nil
}

func (m *MockNotifier) TaskStatusChanged(ctx context.Context, task *Task, actor Person, oldStatus, newStatus Status) error {
	m.statusChangedCalls = append(m.statusChangedCalls, MockNotifierCall{
		TaskID:   task.ID,
		Actor:    actor,
		OldValue: string(oldStatus),
		NewValue: string(newStatus),
	})
	return nil
}

func (m *MockNotifier) TaskDeleted(ctx context.Context, task *Task, actor Person) error {
	m.deletedCalls = append(m.deletedCalls, MockNotifierCall{
		TaskID: task.ID,
		Actor:  actor,
	})
	return nil
}

// ===== 测试用例 =====

// TestCreate_WithAssignees_ShouldNotify 测试创建任务时通知指派人
func TestCreate_WithAssignees_ShouldNotify(t *testing.T) {
	ctx := context.Background()
	repo := NewMockRepository()
	notifier := &MockNotifier{}
	service := NewService(repo, notifier)

	creator := Person{ID: "1", DisplayName: "Creator"}
	assignee := Person{ID: "2", DisplayName: "Assignee"}

	input := CreateInput{
		Title:   "Test Task",
		Creator: creator,
		Assignees: []Person{
			assignee,
			creator, // creator 也在指派人列表中
		},
		Status: StatusPending,
	}

	created, err := service.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if created.ID == "" {
		t.Error("Task ID is empty")
	}

	// 【修复验证】创建任务后应该调用 TaskCreated 通知
	if len(notifier.createdCalls) != 1 {
		t.Errorf("Expected 1 TaskCreated call, got %d", len(notifier.createdCalls))
	}

	call := notifier.createdCalls[0]
	if call.Actor.ID != creator.ID {
		t.Errorf("Wrong creator: got %s, want %s", call.Actor.ID, creator.ID)
	}
}

// TestCreate_WithoutAssignees_ShouldNotNotify 测试没有指派人时不通知
func TestCreate_WithoutAssignees_ShouldNotNotify(t *testing.T) {
	ctx := context.Background()
	repo := NewMockRepository()
	notifier := &MockNotifier{}
	service := NewService(repo, notifier)

	creator := Person{ID: "1", DisplayName: "Creator"}

	input := CreateInput{
		Title:     "Test Task",
		Creator:   creator,
		Assignees: []Person{}, // 没有其他指派人
		Status:    StatusPending,
	}

	_, err := service.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 【修复验证】没有指派人时不应该通知
	if len(notifier.createdCalls) != 0 {
		t.Errorf("Expected 0 TaskCreated calls, got %d", len(notifier.createdCalls))
	}
}

// TestUpdate_WithStatusChange_ShouldNotify 测试状态变化时通知
func TestUpdate_WithStatusChange_ShouldNotify(t *testing.T) {
	ctx := context.Background()
	repo := NewMockRepository()
	notifier := &MockNotifier{}
	service := NewService(repo, notifier)

	// 准备初始任务
	initialTask := &Task{
		ID:        "123",
		Title:     "Test Task",
		Status:    StatusPending,
		CreatedBy: Person{ID: "1", DisplayName: "Creator"},
		Assignees: []Person{
			{ID: "2", DisplayName: "Assignee1"},
		},
	}
	repo.tasks["123"] = initialTask

	// 执行更新
	actor := &Person{ID: "2", DisplayName: "Assignee1"}
	newStatus := StatusCompleted
	updated, err := service.Update(ctx, "123", actor, nil, &newStatus)

	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Status != StatusCompleted {
		t.Errorf("Status not updated: got %v, want %v", updated.Status, StatusCompleted)
	}

	// 【修复验证】状态变化时应该调用 TaskStatusChanged
	if len(notifier.statusChangedCalls) != 1 {
		t.Errorf("Expected 1 TaskStatusChanged call, got %d", len(notifier.statusChangedCalls))
	}

	call := notifier.statusChangedCalls[0]
	if call.OldValue != string(StatusPending) {
		t.Errorf("Wrong old status: got %s, want %s", call.OldValue, StatusPending)
	}
	if call.NewValue != string(StatusCompleted) {
		t.Errorf("Wrong new status: got %s, want %s", call.NewValue, StatusCompleted)
	}
	if call.Actor.ID != "2" {
		t.Errorf("Wrong actor: got %s, want 2", call.Actor.ID)
	}
}

// TestUpdate_WithoutStatusChange_ShouldNotNotify 测试无状态变化不通知
func TestUpdate_WithoutStatusChange_ShouldNotNotify(t *testing.T) {
	ctx := context.Background()
	repo := NewMockRepository()
	notifier := &MockNotifier{}
	service := NewService(repo, notifier)

	// 准备初始任务
	initialTask := &Task{
		ID:        "123",
		Title:     "Test Task",
		Status:    StatusPending,
		CreatedBy: Person{ID: "1", DisplayName: "Creator"},
	}
	repo.tasks["123"] = initialTask

	// 执行更新，只改标题
	actor := &Person{ID: "2", DisplayName: "Updater"}
	newTitle := "Updated Title"
	updated, err := service.Update(ctx, "123", actor, &newTitle, nil)

	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Title != newTitle {
		t.Errorf("Title not updated: got %s, want %s", updated.Title, newTitle)
	}

	// 【修复验证】没有状态变化时不应该通知
	if len(notifier.statusChangedCalls) != 0 {
		t.Errorf("Expected 0 TaskStatusChanged calls, got %d", len(notifier.statusChangedCalls))
	}
}

// TestUpdate_WithoutActor_ShouldNotNotify 测试没有 actor 不通知
func TestUpdate_WithoutActor_ShouldNotNotify(t *testing.T) {
	ctx := context.Background()
	repo := NewMockRepository()
	notifier := &MockNotifier{}
	service := NewService(repo, notifier)

	// 准备初始任务
	initialTask := &Task{
		ID:        "123",
		Title:     "Test Task",
		Status:    StatusPending,
		CreatedBy: Person{ID: "1", DisplayName: "Creator"},
	}
	repo.tasks["123"] = initialTask

	// 执行更新，不传递 actor
	newStatus := StatusCompleted
	updated, err := service.Update(ctx, "123", nil, nil, &newStatus)

	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Status != StatusCompleted {
		t.Errorf("Status not updated: got %v, want %v", updated.Status, StatusCompleted)
	}

	// 【修复验证】没有 actor 不应该通知
	if len(notifier.statusChangedCalls) != 0 {
		t.Errorf("Expected 0 TaskStatusChanged calls, got %d", len(notifier.statusChangedCalls))
	}
}

// TestDelete_WithActor_ShouldNotify 测试删除时通知
func TestDelete_WithActor_ShouldNotify(t *testing.T) {
	ctx := context.Background()
	repo := NewMockRepository()
	notifier := &MockNotifier{}
	service := NewService(repo, notifier)

	// 准备初始任务
	initialTask := &Task{
		ID:        "123",
		Title:     "Test Task",
		Status:    StatusPending,
		CreatedBy: Person{ID: "1", DisplayName: "Creator"},
		Assignees: []Person{
			{ID: "2", DisplayName: "Assignee"},
		},
	}
	repo.tasks["123"] = initialTask

	// 执行删除
	actor := &Person{ID: "2", DisplayName: "Deleter"}
	err := service.Delete(ctx, "123", actor)

	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 【修复验证】删除时应该调用 TaskDeleted
	if len(notifier.deletedCalls) != 1 {
		t.Errorf("Expected 1 TaskDeleted call, got %d", len(notifier.deletedCalls))
	}

	call := notifier.deletedCalls[0]
	if call.Actor.ID != "2" {
		t.Errorf("Wrong actor: got %s, want 2", call.Actor.ID)
	}
}

// TestDelete_WithoutActor_ShouldNotNotify 测试没有 actor 不通知
func TestDelete_WithoutActor_ShouldNotNotify(t *testing.T) {
	ctx := context.Background()
	repo := NewMockRepository()
	notifier := &MockNotifier{}
	service := NewService(repo, notifier)

	// 准备初始任务
	initialTask := &Task{
		ID:        "123",
		Title:     "Test Task",
		Status:    StatusPending,
		CreatedBy: Person{ID: "1", DisplayName: "Creator"},
	}
	repo.tasks["123"] = initialTask

	// 执行删除，不传递 actor
	err := service.Delete(ctx, "123", nil)

	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 【修复验证】没有 actor 不应该通知
	if len(notifier.deletedCalls) != 0 {
		t.Errorf("Expected 0 TaskDeleted calls, got %d", len(notifier.deletedCalls))
	}
}

// TestNotifierError_DoesNotBlockOperation 测试通知错误不阻塞操作
func TestNotifierError_DoesNotBlockOperation(t *testing.T) {
	ctx := context.Background()
	repo := NewMockRepository()

	// 创建会报错的通知器
	errorNotifier := &ErrorNotifier{}
	service := NewService(repo, errorNotifier)

	creator := Person{ID: "1", DisplayName: "Creator"}
	assignee := Person{ID: "2", DisplayName: "Assignee"}

	input := CreateInput{
		Title:     "Test Task",
		Creator:   creator,
		Assignees: []Person{assignee},
		Status:    StatusPending,
	}

	// 【修复验证】即使通知器报错，Create 也应该成功
	created, err := service.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create should succeed even if notifier fails: %v", err)
	}

	if created.ID == "" {
		t.Error("Task should still be created")
	}

	if len(errorNotifier.failedTasks) != 1 {
		t.Errorf("Expected notifier to be called, got %d", len(errorNotifier.failedTasks))
	}
}

// ErrorNotifier 模拟总是报错的通知器
type ErrorNotifier struct {
	failedTasks []string
}

func (e *ErrorNotifier) TaskCreated(ctx context.Context, task *Task, creator Person) error {
	e.failedTasks = append(e.failedTasks, task.ID)
	return errors.New("notification failed")
}

func (e *ErrorNotifier) TaskStatusChanged(ctx context.Context, task *Task, actor Person, oldStatus, newStatus Status) error {
	return errors.New("notification failed")
}

func (e *ErrorNotifier) TaskDeleted(ctx context.Context, task *Task, actor Person) error {
	return errors.New("notification failed")
}
