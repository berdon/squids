package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const currentSchemaVersion = 1

type Task struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	Status      string            `json:"status"`
	Priority    int               `json:"priority,omitempty"`
	IssueType   string            `json:"issue_type,omitempty"`
	Assignee    string            `json:"assignee,omitempty"`
	Owner       string            `json:"owner,omitempty"`
	Labels      []string          `json:"labels,omitempty"`
	Deps        []string          `json:"deps,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CloseReason string            `json:"close_reason,omitempty"`
	CreatedAt   string            `json:"created_at,omitempty"`
	UpdatedAt   string            `json:"updated_at,omitempty"`
	ClosedAt    string            `json:"closed_at,omitempty"`
}

type DBWrap struct {
	Path string
	DB   *sql.DB
}

type CreateInput struct {
	Title       string
	Description string
	IssueType   string
	Priority    int
}

type UpdateInput struct {
	Status      *string
	Assignee    *string
	AddLabels   []string
	SetMetadata map[string]string
	Claim       bool
}

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
CREATE TABLE IF NOT EXISTS counters (
  key TEXT PRIMARY KEY,
  value INTEGER NOT NULL
);
`); err != nil {
		return fmt.Errorf("create counters: %w", err)
	}

	if _, err := tx.Exec(`
INSERT OR IGNORE INTO counters(key,value) VALUES ('issue_seq',0);
`); err != nil {
		return fmt.Errorf("seed counter: %w", err)
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

func EnsureInitialized(db *sql.DB) error {
	if db == nil {
		return errors.New("db is nil")
	}
	row := db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name='tasks'`)
	var n int
	if err := row.Scan(&n); err == nil && n > 0 {
		return nil
	}
	return Init(db)
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

func nextID(db *sql.DB) (string, error) {
	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`INSERT OR IGNORE INTO counters(key,value) VALUES ('issue_seq',0)`); err != nil {
		return "", err
	}
	if _, err := tx.Exec(`UPDATE counters SET value = value + 1 WHERE key='issue_seq'`); err != nil {
		return "", err
	}
	row := tx.QueryRow(`SELECT value FROM counters WHERE key='issue_seq'`)
	var n int
	if err := row.Scan(&n); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return fmt.Sprintf("bd-%d", n), nil
}

func CreateTask(db *sql.DB, in CreateInput) (*Task, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, errors.New("title is required")
	}
	id, err := nextID(db)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.Exec(`INSERT INTO tasks(id,title,description,status,priority,issue_type,labels_json,deps_json,metadata_json,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		id, in.Title, in.Description, "open", in.Priority, in.IssueType, "[]", "[]", "{}", now, now)
	if err != nil {
		return nil, err
	}
	return ShowTask(db, id)
}

func ShowTask(db *sql.DB, id string) (*Task, error) {
	row := db.QueryRow(`SELECT id,title,description,status,priority,issue_type,assignee,owner,labels_json,deps_json,metadata_json,close_reason,created_at,updated_at,closed_at FROM tasks WHERE id=?`, id)
	var t Task
	var labels, deps, metadata string
	if err := row.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.IssueType, &t.Assignee, &t.Owner, &labels, &deps, &metadata, &t.CloseReason, &t.CreatedAt, &t.UpdatedAt, &t.ClosedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("issue not found: %s", id)
		}
		return nil, err
	}
	_ = json.Unmarshal([]byte(labels), &t.Labels)
	_ = json.Unmarshal([]byte(deps), &t.Deps)
	t.Metadata = map[string]string{}
	_ = json.Unmarshal([]byte(metadata), &t.Metadata)
	return &t, nil
}

func ListTasks(db *sql.DB) ([]Task, error) {
	rows, err := db.Query(`SELECT id,title,description,status,priority,issue_type,assignee,owner,labels_json,deps_json,metadata_json,close_reason,created_at,updated_at,closed_at FROM tasks ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Task, 0)
	for rows.Next() {
		var t Task
		var labels, deps, metadata string
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.IssueType, &t.Assignee, &t.Owner, &labels, &deps, &metadata, &t.CloseReason, &t.CreatedAt, &t.UpdatedAt, &t.ClosedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(labels), &t.Labels)
		_ = json.Unmarshal([]byte(deps), &t.Deps)
		t.Metadata = map[string]string{}
		_ = json.Unmarshal([]byte(metadata), &t.Metadata)
		out = append(out, t)
	}
	return out, nil
}

func UpdateTask(db *sql.DB, id string, in UpdateInput) (*Task, error) {
	t, err := ShowTask(db, id)
	if err != nil {
		return nil, err
	}
	if in.Status != nil {
		t.Status = *in.Status
	}
	if in.Assignee != nil {
		t.Assignee = *in.Assignee
	}
	if in.Claim {
		t.Status = "in_progress"
	}
	if len(in.AddLabels) > 0 {
		existing := map[string]bool{}
		for _, l := range t.Labels {
			existing[l] = true
		}
		for _, l := range in.AddLabels {
			if l != "" && !existing[l] {
				t.Labels = append(t.Labels, l)
				existing[l] = true
			}
		}
	}
	if t.Metadata == nil {
		t.Metadata = map[string]string{}
	}
	for k, v := range in.SetMetadata {
		if strings.TrimSpace(k) != "" {
			t.Metadata[k] = v
		}
	}

	if upstream := strings.TrimSpace(t.Metadata["upstream"]); upstream != "" {
		exists := false
		for _, d := range t.Deps {
			if d == upstream {
				exists = true
				break
			}
		}
		if !exists {
			t.Deps = append(t.Deps, upstream)
		}
	}

	labelsRaw, _ := json.Marshal(t.Labels)
	depsRaw, _ := json.Marshal(t.Deps)
	metaRaw, _ := json.Marshal(t.Metadata)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.Exec(`UPDATE tasks SET status=?,assignee=?,labels_json=?,deps_json=?,metadata_json=?,updated_at=? WHERE id=?`, t.Status, t.Assignee, string(labelsRaw), string(depsRaw), string(metaRaw), now, id)
	if err != nil {
		return nil, err
	}
	return ShowTask(db, id)
}

func CloseTask(db *sql.DB, id, reason string) (*Task, error) {
	if _, err := ShowTask(db, id); err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`UPDATE tasks SET status='closed',close_reason=?,closed_at=?,updated_at=? WHERE id=?`, reason, now, now, id)
	if err != nil {
		return nil, err
	}
	return ShowTask(db, id)
}
