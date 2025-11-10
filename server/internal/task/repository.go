package task

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
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
		       u.id, u.display_name, COALESCE(u.username, ''), COALESCE(u.avatar_url, '')
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
		       u.id, u.display_name, COALESCE(u.username, ''), COALESCE(u.avatar_url, '')
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

// Create 新增任务并写入关联的指派人与 Telegram 消息引用。
func (r *Repository) Create(ctx context.Context, input CreateInput) (*Task, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	creatorID, err := strconv.ParseInt(input.Creator.ID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse creator id %q: %w", input.Creator.ID, err)
	}
	if err = upsertUser(ctx, tx, input.Creator); err != nil {
		return nil, err
	}

	normalizedAssignees, err := normalizePeople(input.Assignees)
	if err != nil {
		return nil, err
	}
	for _, assignee := range normalizedAssignees {
		if err = upsertUser(ctx, tx, assignee.Person); err != nil {
			return nil, err
		}
	}

	description := stringPtrToNull(input.Description)
	sourceURL := stringPtrToNull(input.SourceMessageURL)
	status := string(input.Status)
	if status == "" {
		status = string(StatusPending)
	}

	var taskID int64
	if err = tx.QueryRowContext(ctx, `
		INSERT INTO tasks (title, description, status, creator_id, source_message_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		input.Title,
		description,
		status,
		creatorID,
		sourceURL,
	).Scan(&taskID); err != nil {
		return nil, err
	}

	for _, assignee := range normalizedAssignees {
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO task_assignees (task_id, user_id)
			VALUES ($1, $2)
			ON CONFLICT (task_id, user_id) DO NOTHING`,
			taskID, assignee.ID,
		); err != nil {
			return nil, err
		}
	}

	if input.TelegramMessage != nil &&
		input.TelegramMessage.ChatID != 0 &&
		input.TelegramMessage.MessageID != 0 {
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO telegram_messages (task_id, chat_id, message_id)
			VALUES ($1, $2, $3)`,
			taskID,
			input.TelegramMessage.ChatID,
			input.TelegramMessage.MessageID,
		); err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return r.Get(ctx, strconv.FormatInt(taskID, 10))
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
		SELECT u.id, u.display_name, COALESCE(u.username, ''), COALESCE(u.avatar_url, '')
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
		var displayName, username, avatar string
		if err := rows.Scan(&id, &displayName, &username, &avatar); err != nil {
			return nil, err
		}
		assignees = append(assignees, Person{
			ID:          strconv.FormatInt(id, 10),
			DisplayName: displayName,
			Username:    username,
			AvatarURL:   avatar,
		})
	}
	return assignees, rows.Err()
}

func scanTask(scanner interface {
	Scan(dest ...any) error
}) (*Task, error) {
	var (
		id, creatorID                                            int64
		title, description, status, creatorName, creatorUsername string
		creatorAvatar                                            string
		sourceURL                                                sql.NullString
		createdAt                                                time.Time
	)
	if err := scanner.Scan(
		&id,
		&title,
		&description,
		&status,
		&createdAt,
		&sourceURL,
		&creatorID,
		&creatorName,
		&creatorUsername,
		&creatorAvatar,
	); err != nil {
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
			AvatarURL:   creatorAvatar,
		},
		SourceMessage: sourceMessage,
		Permissions: Permissions{
			CanEdit:     true,
			CanComplete: true,
			CanDelete:   true,
		},
	}, nil
}

type normalizedPerson struct {
	Person Person
	ID     int64
}

func normalizePeople(people []Person) ([]normalizedPerson, error) {
	normalized := make([]normalizedPerson, 0, len(people))
	seen := make(map[string]struct{})
	for _, person := range people {
		if _, exists := seen[person.ID]; exists {
			continue
		}
		seen[person.ID] = struct{}{}
		if strings.TrimSpace(person.ID) == "" {
			continue
		}
		userID, err := strconv.ParseInt(person.ID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse assignee id %q: %w", person.ID, err)
		}
		normalized = append(normalized, normalizedPerson{
			Person: person,
			ID:     userID,
		})
	}
	return normalized, nil
}

func upsertUser(ctx context.Context, tx *sql.Tx, person Person) error {
	userID, err := strconv.ParseInt(person.ID, 10, 64)
	if err != nil {
		return fmt.Errorf("parse user id %q: %w", person.ID, err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO users (id, display_name, username, avatar_url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE
		SET display_name = EXCLUDED.display_name,
		    username = COALESCE(EXCLUDED.username, users.username),
		    avatar_url = COALESCE(EXCLUDED.avatar_url, users.avatar_url)`,
		userID,
		person.DisplayName,
		stringToNull(person.Username),
		stringToNull(person.AvatarURL),
	)
	return err
}

func stringPtrToNull(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	if strings.TrimSpace(*value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func stringToNull(value string) sql.NullString {
	if strings.TrimSpace(value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}
