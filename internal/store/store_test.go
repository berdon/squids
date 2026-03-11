package store

import (
	"path/filepath"
	"regexp"
	"testing"
)

func openTestDB(t *testing.T) (*DBWrap, func()) {
	t.Helper()
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "data", "tasks.sqlite")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := Init(db); err != nil {
		t.Fatalf("init: %v", err)
	}
	return &DBWrap{Path: dbPath, DB: db}, func() { _ = db.Close() }
}

func TestInitAndVersion(t *testing.T) {
	w, done := openTestDB(t)
	defer done()

	v, err := CurrentVersion(w.DB)
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if v != 1 {
		t.Fatalf("expected version 1, got %d", v)
	}

	if err := Ping(w.DB); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestDefaultDBPath(t *testing.T) {
	got := DefaultDBPath("/tmp/x")
	want := "/tmp/x/.sq/tasks.sqlite"
	if got != want {
		t.Fatalf("want %s got %s", want, got)
	}
}

func TestEnsureInitializedIdempotent(t *testing.T) {
	w, done := openTestDB(t)
	defer done()
	if err := EnsureInitialized(w.DB); err != nil {
		t.Fatalf("ensure initialized: %v", err)
	}
}

func TestLabelsMetadataAndDepsParity(t *testing.T) {
	w, done := openTestDB(t)
	defer done()

	created, err := CreateTask(w.DB, CreateInput{Title: "Task A", IssueType: "task", Priority: 1})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if ok, _ := regexp.MatchString(`^bd-[0-9a-z]{3,8}$`, created.ID); !ok {
		t.Fatalf("expected beads-style hash id (bd-[0-9a-z]{3,8}), got %s", created.ID)
	}
	dep, err := CreateTask(w.DB, CreateInput{Title: "Task B", IssueType: "task", Priority: 2})
	if err != nil {
		t.Fatalf("create dep: %v", err)
	}

	updated, err := UpdateTask(w.DB, created.ID, UpdateInput{
		AddLabels: []string{"shop:forge", "type:mail", "shop:forge"},
		SetMetadata: map[string]string{
			"upstream": dep.ID,
		},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	if len(updated.Labels) != 2 {
		t.Fatalf("expected 2 unique labels, got %d (%v)", len(updated.Labels), updated.Labels)
	}
	if updated.Metadata["upstream"] != dep.ID {
		t.Fatalf("expected metadata upstream=%s got %s", dep.ID, updated.Metadata["upstream"])
	}
	if len(updated.Deps) != 1 || updated.Deps[0] != dep.ID {
		t.Fatalf("expected deps [%s], got %v", dep.ID, updated.Deps)
	}
}
