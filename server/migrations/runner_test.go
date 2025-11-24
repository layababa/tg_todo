package migrations

import (
	"io/fs"
	"testing"
)

func TestRunReturnsErrorWhenDBNil(t *testing.T) {
	err := Run(nil)
	if err == nil {
		t.Fatal("expected error when db is nil")
	}
	if err != ErrNilDatabase {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEmbeddedFilesAvailable(t *testing.T) {
	entries, err := fs.ReadDir(Files, "sql")
	if err != nil {
		t.Fatalf("failed to read embedded sql dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected embedded migrations, got none")
	}
}
