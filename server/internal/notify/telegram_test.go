package notify

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/zz/tg_todo/server/internal/task"
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestNotifier(handler func(req *http.Request) (*http.Response, error)) *TelegramNotifier {
	return &TelegramNotifier{
		token:   "fake-token",
		apiBase: "https://api.telegram.test",
		client:  &http.Client{Transport: roundTripperFunc(handler)},
	}
}

func okResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		Header:     make(http.Header),
	}
}

func TestTelegramNotifier_TaskStatusChanged_SendsToRecipients(t *testing.T) {
	var mu sync.Mutex
	var paths []string
	notifier := newTestNotifier(func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		paths = append(paths, req.URL.Path)
		mu.Unlock()
		return okResponse(), nil
	})
	taskData := &task.Task{
		ID:            "42",
		Title:         "集成测试",
		SourceMessage: "https://t.me/c/123/456",
		CreatedBy: task.Person{
			ID:          "100",
			DisplayName: "创建人",
		},
		Assignees: []task.Person{
			{ID: "200", DisplayName: "执行人"},
		},
	}
	actor := task.Person{ID: "200", DisplayName: "执行人"}

	err := notifier.TaskStatusChanged(context.Background(), taskData, actor, task.StatusPending, task.StatusCompleted)
	if err != nil {
		t.Fatalf("TaskStatusChanged returned error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(paths) != 1 {
		t.Fatalf("expected 1 sendMessage call, got %d", len(paths))
	}
	if !strings.Contains(paths[0], "sendMessage") {
		t.Errorf("unexpected request path: %s", paths[0])
	}
}

func TestTelegramNotifier_TaskCreated_NotifiesAssigneesOnly(t *testing.T) {
	var mu sync.Mutex
	var payloads []string
	notifier := newTestNotifier(func(req *http.Request) (*http.Response, error) {
		if err := req.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		mu.Lock()
		payloads = append(payloads, req.Form.Encode())
		mu.Unlock()
		return okResponse(), nil
	})
	taskData := &task.Task{
		ID:    "99",
		Title: "新任务",
		Assignees: []task.Person{
			{ID: "301", DisplayName: "前端"},
			{ID: "302", DisplayName: "后端"},
		},
	}
	creator := task.Person{ID: "100", DisplayName: "创建人"}

	err := notifier.TaskCreated(context.Background(), taskData, creator)
	if err != nil {
		t.Fatalf("TaskCreated returned error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(payloads) != 2 {
		t.Fatalf("expected 2 sendMessage calls, got %d", len(payloads))
	}
	if !strings.Contains(payloads[0], "chat_id=301") || !strings.Contains(payloads[1], "chat_id=302") {
		t.Errorf("unexpected payloads: %#v", payloads)
	}
}
