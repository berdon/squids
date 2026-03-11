package cli

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/berdon/squids/internal/store"
)

func seedSourceDB(t *testing.T, dbPath string) {
	t.Helper()
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	defer db.Close()
	if err := store.Init(db); err != nil {
		t.Fatalf("init source: %v", err)
	}

	_, err = db.Exec(`INSERT INTO tasks(id,title,description,status,priority,issue_type,assignee,owner,labels_json,deps_json,metadata_json,close_reason,created_at,updated_at,closed_at)
VALUES('bd-1','Imported task','desc','open',1,'task','alice','alice','["triage"]','[]','{"k":"v"}','','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z','')`)
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}
	_, _ = db.Exec(`INSERT INTO comments(issue_id,author,body,created_at) VALUES('bd-1','alice','hello','2026-01-01T00:00:01Z')`)
}

func openCount(t *testing.T, dbPath, table string) int {
	t.Helper()
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open count db: %v", err)
	}
	defer db.Close()
	if err := store.EnsureInitialized(db); err != nil {
		t.Fatalf("ensure init: %v", err)
	}
	row := db.QueryRow(`SELECT COUNT(1) FROM ` + table)
	var n int
	if err := row.Scan(&n); err != nil {
		t.Fatalf("scan count: %v", err)
	}
	return n
}

func TestImportBeadsCommand_ImportsAndIsIdempotent(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "target.sqlite")
	source := filepath.Join(tmp, "source.sqlite")
	seedSourceDB(t, source)

	old := os.Getenv("SQ_DB_PATH")
	_ = os.Setenv("SQ_DB_PATH", target)
	defer func() { _ = os.Setenv("SQ_DB_PATH", old) }()

	if code := cmdImportBeads([]string{"--source", source, "--json"}); code != 0 {
		t.Fatalf("first import failed: %d", code)
	}

	if got := openCount(t, target, "tasks"); got != 1 {
		t.Fatalf("expected 1 imported task got %d", got)
	}

	if code := cmdImportBeads([]string{"--source", source, "--json"}); code != 0 {
		t.Fatalf("second import failed: %d", code)
	}
	if got := openCount(t, target, "tasks"); got != 1 {
		t.Fatalf("expected idempotent task count 1 got %d", got)
	}
}

func TestImportBeadsCommand_DryRunNoWrites(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "target.sqlite")
	source := filepath.Join(tmp, "source.sqlite")
	seedSourceDB(t, source)

	old := os.Getenv("SQ_DB_PATH")
	_ = os.Setenv("SQ_DB_PATH", target)
	defer func() { _ = os.Setenv("SQ_DB_PATH", old) }()

	if code := cmdImportBeads([]string{"--source", source, "--dry-run", "--json"}); code != 0 {
		t.Fatalf("dry run failed: %d", code)
	}
	if got := openCount(t, target, "tasks"); got != 0 {
		t.Fatalf("expected no writes in dry-run got %d tasks", got)
	}
}

func TestImportBeadsCommand_RejectsMissingTasksTable(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "bad.sqlite")
	db, err := sql.Open("sqlite3", source)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	_, _ = db.Exec(`CREATE TABLE nope (id TEXT)`)
	_ = db.Close()

	if code := cmdImportBeads([]string{"--source", source, "--json"}); code != 2 {
		t.Fatalf("expected validation code 2 got %d", code)
	}
}

