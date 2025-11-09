package task

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"
)

// Repository 提供任务在 Postgres 中的读写操作。
type Repository struct {
	db *sql.DB
}

// NewRepository 返回任务存储层实例。
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ErrNotFound 表示任务不存在。
var ErrNotFound = errors.New("task not found")

// List 返回全部任务，包含指派人列表。
func (r *Repository) List(ctx context.Context) ([]*Task, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id, t.title, COALESCE(t.description, ''), t.status, t.created_at,
		       t.source_message_url,
		       u.id, u.display_name, COALESCE(u.username, '')
		  FROM tasks t
		  JOIN users u ON u.id = t.creator_id
		 ORDER BY t.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		assignees, err := r.fetchAssignees(ctx, task.ID)
		if err != nil {
			return nil, err
		}
		task.Assignees = assignees
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

// Get 获取单个任务。
func (r *Repository) Get(ctx context.Context, id string) (*Task, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT t.id, t.title, COALESCE(t.description, ''), t.status, t.created_at,
		       t.source_message_url,
		       u.id, u.display_name, COALESCE(u.username, '')
		  FROM tasks t
		  JOIN users u ON u.id = t.creator_id
		 WHERE t.id = $1`, id)

	task, err := scanTask(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	assignees, err := r.fetchAssignees(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	task.Assignees = assignees
	return task, nil
}

// Update 修改标题或状态。
func (r *Repository) Update(ctx context.Context, id string, title *string, status *Status) (*Task, error) {
	titleVal := ""
	if title != nil {
		titleVal = *title
	}
	statusVal := ""
	if status != nil {
		statusVal = string(*status)
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE tasks
		   SET title = COALESCE(NULLIF($2, ''), title),
		       status = CASE WHEN $3 = '' THEN status ELSE $3 END,
		       updated_at = NOW()
		 WHERE id = $1`, id, titleVal, statusVal)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, id)
}

// Delete 移除任务。
func (r *Repository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) fetchAssignees(ctx context.Context, taskID string) ([]Person, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.display_name, COALESCE(u.username, '')
		  FROM task_assignees ta
		  JOIN users u ON u.id = ta.user_id
		 WHERE ta.task_id = $1
		 ORDER BY u.display_name`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignees []Person
	for rows.Next() {
		var id int64
		var displayName, username string
		if err := rows.Scan(&id, &displayName, &username); err != nil {
			return nil, err
		}
		assignees = append(assignees, Person{
			ID:          strconv.FormatInt(id, 10),
			DisplayName: displayName,
			Username:    username,
		})
	}
	return assignees, rows.Err()
}

func scanTask(scanner interface {
	Scan(dest ...any) error
}) (*Task, error) {
	var (
		id, creatorID                               int64
		title, description, status, creatorName, creatorUsername string
		sourceURL                                   sql.NullString
		createdAt                                   time.Time
	)
	if err := scanner.Scan(&id, &title, &description, &status, &createdAt, &sourceURL, &creatorID, &creatorName, &creatorUsername); err != nil {
		return nil, err
	}

	sourceMessage := ""
	if sourceURL.Valid {
		sourceMessage = sourceURL.String
	}

	return &Task{
		ID:          strconv.FormatInt(id, 10),
		Title:       title,
		Description: description,
		Status:      Status(status),
		CreatedAt:   createdAt,
		CreatedBy: Person{
			ID:          strconv.FormatInt(creatorID, 10),
			DisplayName: creatorName,
			Username:    creatorUsername,
		},
		SourceMessage: sourceMessage,
		Permissions: Permissions{
			CanEdit:     true,
			CanComplete: true,
			CanDelete:   true,
		},
	}, nil
}
