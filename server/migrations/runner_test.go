package migrations

import "testing"

func TestRunReturnsErrorWhenDBNil(t *testing.T) {
	err := Run(nil)
	if err == nil {
		t.Fatal("expected error when db is nil")
	}
	if err != ErrNilDatabase {
		t.Fatalf("unexpected error: %v", err)
	}
}
