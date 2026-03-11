package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitAndVersion(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "data", "tasks.sqlite")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := Init(db); err != nil {
		t.Fatalf("init: %v", err)
	}

	v, err := CurrentVersion(db)
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if v != 1 {
		t.Fatalf("expected version 1, got %d", v)
	}

	if err := Ping(db); err != nil {
		t.Fatalf("ping: %v", err)
	}

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("db file not created: %v", err)
	}
}

func TestDefaultDBPath(t *testing.T) {
	got := DefaultDBPath("/tmp/x")
	want := "/tmp/x/.sq/tasks.sqlite"
	if got != want {
		t.Fatalf("want %s got %s", want, got)
	}
}
