package task

import "context"

// Notifier 定义任务生命周期事件通知接口。
type Notifier interface {
	TaskCreated(ctx context.Context, task *Task, creator Person) error
	TaskStatusChanged(ctx context.Context, task *Task, actor Person, oldStatus, newStatus Status) error
	TaskDeleted(ctx context.Context, task *Task, actor Person) error
}
