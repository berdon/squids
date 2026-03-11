package store

import (
	"database/sql"
	"testing"
)

func TestShowUpdateCloseReopenMissing(t *testing.T) {
	w, done := openTestDB(t)
	defer done()

	if _, err := ShowTask(w.DB, "bd-missing"); err == nil {
		t.Fatalf("expected missing show error")
	}
	if _, err := UpdateTask(w.DB, "bd-missing", UpdateInput{}); err == nil {
		t.Fatalf("expected missing update error")
	}
	if _, err := CloseTask(w.DB, "bd-missing", "x"); err == nil {
		t.Fatalf("expected missing close error")
	}
	if _, err := ReopenTask(w.DB, "bd-missing"); err == nil {
		t.Fatalf("expected missing reopen error")
	}
}

func TestCommentAndDependencyMissingPaths(t *testing.T) {
	w, done := openTestDB(t)
	defer done()

	if _, err := AddComment(w.DB, "bd-missing", "me", "hello"); err == nil {
		t.Fatalf("expected add comment missing issue error")
	}
	if _, err := ListComments(w.DB, "bd-missing"); err == nil {
		t.Fatalf("expected list comments missing issue error")
	}
	if _, err := ListDependencies(w.DB, "bd-missing"); err != nil {
		t.Fatalf("list deps for missing should still be empty list, got err=%v", err)
	}
}

func TestCurrentVersionNoTable(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open mem: %v", err)
	}
	defer db.Close()
	if _, err := CurrentVersion(db); err == nil {
		t.Fatalf("expected current version error without schema_migrations")
	}
}
