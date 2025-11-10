package bot

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func httpOK(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestBotCollectAssigneesResolvesMentions(t *testing.T) {
	var gotChatMember bool
	b := New("token", "https://api.test", nil)
	b.client = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.Path, "getChatMember") {
				return nil, fmt.Errorf("unexpected path: %s", req.URL.Path)
			}
			if err := req.ParseForm(); err != nil {
				return nil, err
			}
			if req.PostForm.Get("user_id") != "@fe_lead" {
				return nil, fmt.Errorf("unexpected user_id: %s", req.PostForm.Get("user_id"))
			}
			gotChatMember = true
			return httpOK(`{"ok":true,"result":{"status":"member","user":{"id":2000,"is_bot":false,"first_name":"FE","username":"fe_lead"}}}`), nil
		}),
	}
	msg := &Message{
		Text: "请处理 @fe_lead 的待办",
		Entities: []MessageEntity{
			{Type: "mention", Offset: 4, Length: len("@fe_lead")},
		},
		From: &User{ID: 1000, FirstName: "Creator"},
		Chat: Chat{ID: -1001234, Type: "supergroup"},
	}

	creator := userToPerson(msg.From)
	assignees := b.collectAssignees(context.Background(), msg, creator)

	if !gotChatMember {
		t.Fatal("expected getChatMember to be called")
	}
	if len(assignees) != 2 {
		t.Fatalf("expected 2 assignees (creator + mention), got %d", len(assignees))
	}
	if assignees[1].Username != "fe_lead" {
		t.Errorf("expected resolved username fe_lead, got %s", assignees[1].Username)
	}
}

func TestBotSendMessageHitsTelegramAPI(t *testing.T) {
	var mu sync.Mutex
	var form url.Values
	b := New("token", "https://api.test", nil)
	b.client = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.Path, "sendMessage") {
				return nil, fmt.Errorf("unexpected path: %s", req.URL.Path)
			}
			if err := req.ParseForm(); err != nil {
				return nil, err
			}
			mu.Lock()
			form = req.PostForm
			mu.Unlock()
			return httpOK(`{"ok":true}`), nil
		}),
	}
	if err := b.sendMessage(context.Background(), 12345, "hello world"); err != nil {
		t.Fatalf("sendMessage returned error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if form.Get("chat_id") != "12345" || form.Get("text") != "hello world" {
		t.Fatalf("unexpected form payload: %v", form)
	}
}

func TestBotFetchUpdates(t *testing.T) {
	b := New("token", "https://api.test", nil)
	b.client = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.Path, "getUpdates") {
				return nil, fmt.Errorf("unexpected path: %s", req.URL.Path)
			}
			return httpOK(`{"ok":true,"result":[{"update_id":1,"message":{"message_id":10,"text":"/tasks","chat":{"id":1,"type":"private"},"from":{"id":2}}}]}`), nil
		}),
	}
	updates, err := b.fetchUpdates(context.Background())
	if err != nil {
		t.Fatalf("fetchUpdates returned error: %v", err)
	}
	if len(updates) != 1 || updates[0].UpdateID != 1 {
		t.Fatalf("unexpected updates: %#v", updates)
	}
}
