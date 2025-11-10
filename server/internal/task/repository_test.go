package task

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"time"
)

type fakeScanner struct {
	values []any
}

func (f *fakeScanner) Scan(dest ...any) error {
	if len(dest) != len(f.values) {
		return fmt.Errorf("want %d dests, got %d", len(f.values), len(dest))
	}
	for i, d := range dest {
		rv := reflect.ValueOf(d)
		if rv.Kind() != reflect.Ptr {
			return fmt.Errorf("dest %d not pointer", i)
		}
		value := reflect.ValueOf(f.values[i])
		rv.Elem().Set(value)
	}
	return nil
}

func TestScanTask(t *testing.T) {
	now := time.Now().UTC()
	sourceURL := sql.NullString{String: "https://t.me/c/123/456", Valid: true}

	scanner := &fakeScanner{values: []any{
		int64(1),                      // id
		"撰写 API 文档",                   // title
		"补齐字段",                        // description
		"pending",                     // status
		now,                           // created_at
		sourceURL,                     // source_message_url
		int64(200),                    // creator_id
		"创建人",                         // creator name
		"creator_username",            // creator username
		"https://avatar.test/creator", // creator avatar
	}}

	task, err := scanTask(scanner)
	if err != nil {
		t.Fatalf("scanTask failed: %v", err)
	}

	if task.ID != "1" || task.Title != "撰写 API 文档" {
		t.Fatalf("unexpected task: %#v", task)
	}
	if task.SourceMessage != sourceURL.String {
		t.Errorf("source message mismatch, got %s", task.SourceMessage)
	}
	if task.CreatedBy.AvatarURL != "https://avatar.test/creator" {
		t.Errorf("creator avatar mismatch: %s", task.CreatedBy.AvatarURL)
	}
}

func TestNormalizePeople(t *testing.T) {
	people := []Person{
		{ID: "100", DisplayName: "Creator"},
		{ID: "200", DisplayName: "Assignee"},
		{ID: "200", DisplayName: "Duplicate"},
		{ID: " ", DisplayName: "Invalid"},
	}

	normalized, err := normalizePeople(people)
	if err != nil {
		t.Fatalf("normalizePeople error: %v", err)
	}
	if len(normalized) != 2 {
		t.Fatalf("expected 2 people, got %d", len(normalized))
	}
	if normalized[1].ID != 200 || normalized[1].Person.DisplayName != "Assignee" {
		t.Errorf("unexpected normalized person: %#v", normalized[1])
	}
}

func TestStringPtrToNull(t *testing.T) {
	value := "hello"
	if res := stringPtrToNull(&value); !res.Valid || res.String != "hello" {
		t.Fatalf("expected valid sql.NullString, got %+v", res)
	}
	if res := stringPtrToNull(nil); res.Valid {
		t.Fatalf("expected invalid null string, got %+v", res)
	}
}
