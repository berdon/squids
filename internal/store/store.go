package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const currentSchemaVersion = 1

// Open opens a SQLite database handle with WAL + busy timeout configured.
func Open(dbPath string) (*sql.DB, error) {
	if dbPath == "" {
		return nil, errors.New("db path required")
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=1", dbPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set wal mode: %w", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout=5000;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set busy_timeout: %w", err)
	}

	return db, nil
}

func DefaultDBPath(cwd string) string {
	return filepath.Join(cwd, ".sq", "tasks.sqlite")
}

func Init(db *sql.DB) error {
	if db == nil {
		return errors.New("db is nil")
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);
`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	if _, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  description TEXT DEFAULT '',
  status TEXT NOT NULL,
  priority INTEGER,
  issue_type TEXT,
  assignee TEXT DEFAULT '',
  owner TEXT DEFAULT '',
  labels_json TEXT DEFAULT '[]',
  deps_json TEXT DEFAULT '[]',
  metadata_json TEXT DEFAULT '{}',
  close_reason TEXT DEFAULT '',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  closed_at TEXT DEFAULT ''
);
`); err != nil {
		return fmt.Errorf("create tasks: %w", err)
	}

	if _, err := tx.Exec(`
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
`); err != nil {
		return fmt.Errorf("create status index: %w", err)
	}

	if _, err := tx.Exec(`
INSERT OR IGNORE INTO schema_migrations(version) VALUES (?);
`, currentSchemaVersion); err != nil {
		return fmt.Errorf("insert schema version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit schema init: %w", err)
	}
	return nil
}

func CurrentVersion(db *sql.DB) (int, error) {
	if db == nil {
		return 0, errors.New("db is nil")
	}
	row := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`)
	var v int
	if err := row.Scan(&v); err != nil {
		return 0, fmt.Errorf("scan version: %w", err)
	}
	return v, nil
}

func Ping(db *sql.DB) error {
	if db == nil {
		return errors.New("db is nil")
	}
	return db.Ping()
}
