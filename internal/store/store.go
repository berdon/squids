package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gitea/auhanson/squids/internal/idgen"
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
	Creator     string
}

type UpdateInput struct {
	Status      *string
	Assignee    *string
	AddLabels   []string
	SetMetadata map[string]string
	Claim       bool
}

// Open configures a SQLite handle for local multi-process use.
//
// Separation of concerns: CLI code should not manage low-level SQLite pragmas;
// storage defaults are centralized here.
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
	pragmas := []struct {
		q   string
		ctx string
	}{
		{"PRAGMA journal_mode=WAL;", "set wal mode"},
		{"PRAGMA busy_timeout=5000;", "set busy_timeout"},
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p.q); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("%s: %w", p.ctx, err)
		}
	}
	return db, nil
}

func DefaultDBPath(cwd string) string {
	return filepath.Join(cwd, ".sq", "tasks.sqlite")
}

// Init creates core schema/tables and seeds required config defaults.
func execTx(tx *sql.Tx, query, context string, args ...any) error {
	if _, err := tx.Exec(query, args...); err != nil {
		return fmt.Errorf("%s: %w", context, err)
	}
	return nil
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

	stmts := []struct {
		q   string
		ctx string
	}{
		{`
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);
`, "create schema_migrations"},
		{`
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
`, "create tasks"},
		{`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);`, "create status index"},
		{`
CREATE TABLE IF NOT EXISTS dependencies (
  issue_id TEXT NOT NULL,
  depends_on_id TEXT NOT NULL,
  dep_type TEXT NOT NULL DEFAULT 'blocks',
  PRIMARY KEY(issue_id, depends_on_id, dep_type)
);
`, "create dependencies"},
		{`
CREATE TABLE IF NOT EXISTS comments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  issue_id TEXT NOT NULL,
  author TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
`, "create comments"},
		{`
CREATE TABLE IF NOT EXISTS config (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
`, "create config"},
		{`INSERT OR IGNORE INTO config(key,value) VALUES ('id_prefix','bd')`, "seed id_prefix"},
		{`INSERT OR IGNORE INTO config(key,value) VALUES ('issue_id_mode','hash')`, "seed issue_id_mode"},
		{`INSERT OR IGNORE INTO config(key,value) VALUES ('min_hash_length','3')`, "seed min_hash_length"},
		{`INSERT OR IGNORE INTO config(key,value) VALUES ('max_hash_length','8')`, "seed max_hash_length"},
		{`INSERT OR IGNORE INTO config(key,value) VALUES ('max_collision_prob','0.25')`, "seed max_collision_prob"},
	}
	for _, s := range stmts {
		if err := execTx(tx, s.q, s.ctx); err != nil {
			return err
		}
	}
	if err := execTx(tx, `INSERT OR IGNORE INTO schema_migrations(version) VALUES (?);`, "insert schema version", currentSchemaVersion); err != nil {
		return err
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

func getConfigValue(db *sql.DB, key, fallback string) string {
	row := db.QueryRow(`SELECT value FROM config WHERE key=?`, key)
	var val string
	if err := row.Scan(&val); err != nil || strings.TrimSpace(val) == "" {
		return fallback
	}
	return val
}

func collisionProbability(numIssues int, idLength int) float64 {
	totalPossibilities := math.Pow(36.0, float64(idLength))
	exponent := -float64(numIssues*numIssues) / (2.0 * totalPossibilities)
	return 1.0 - math.Exp(exponent)
}

func computeAdaptiveLength(numIssues, minLength, maxLength int, maxCollisionProb float64) int {
	for length := minLength; length <= maxLength; length++ {
		if collisionProbability(numIssues, length) <= maxCollisionProb {
			return length
		}
	}
	return maxLength
}

func nextCounterID(db *sql.DB, prefix string) (string, error) {
	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()
	steps := []struct {
		q    string
		args []any
	}{
		{`CREATE TABLE IF NOT EXISTS issue_counter (prefix TEXT PRIMARY KEY, last_id INTEGER NOT NULL DEFAULT 0)`, nil},
		{`INSERT OR IGNORE INTO issue_counter(prefix,last_id) VALUES (?,0)`, []any{prefix}},
		{`UPDATE issue_counter SET last_id = last_id + 1 WHERE prefix=?`, []any{prefix}},
	}
	for _, s := range steps {
		if _, err := tx.Exec(s.q, s.args...); err != nil {
			return "", err
		}
	}
	var n int
	if err := tx.QueryRow(`SELECT last_id FROM issue_counter WHERE prefix=?`, prefix).Scan(&n); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%d", prefix, n), nil
}

// CreateTask creates a task-like issue record and returns the stored entity.
func CreateTask(db *sql.DB, in CreateInput) (*Task, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, errors.New("title is required")
	}
	prefix := getConfigValue(db, "id_prefix", "bd")
	idMode := getConfigValue(db, "issue_id_mode", "hash")
	creator := in.Creator
	if strings.TrimSpace(creator) == "" {
		creator = "unknown"
	}

	nowTs := time.Now().UTC()
	now := nowTs.Format(time.RFC3339)

	if idMode == "counter" {
		id, err := nextCounterID(db, prefix)
		if err != nil {
			return nil, err
		}
		_, err = db.Exec(`INSERT INTO tasks(id,title,description,status,priority,issue_type,labels_json,deps_json,metadata_json,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
			id, in.Title, in.Description, "open", in.Priority, in.IssueType, "[]", "[]", "{}", now, now)
		if err != nil {
			return nil, err
		}
		return ShowTask(db, id)
	}

	minLen, _ := strconv.Atoi(getConfigValue(db, "min_hash_length", "3"))
	maxLen, _ := strconv.Atoi(getConfigValue(db, "max_hash_length", "8"))
	maxProb, _ := strconv.ParseFloat(getConfigValue(db, "max_collision_prob", "0.25"), 64)
	if minLen < 3 {
		minLen = 3
	}
	if maxLen > 8 {
		maxLen = 8
	}
	if maxLen < minLen {
		maxLen = minLen
	}

	var numIssues int
	_ = db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE id LIKE ?`, prefix+"-%").Scan(&numIssues)
	baseLen := computeAdaptiveLength(numIssues, minLen, maxLen, maxProb)

	for length := baseLen; length <= maxLen; length++ {
		for nonce := 0; nonce < 10; nonce++ {
			id := idgen.GenerateHashID(prefix, in.Title, in.Description, creator, nowTs, length, nonce)
			_, err := db.Exec(`INSERT INTO tasks(id,title,description,status,priority,issue_type,labels_json,deps_json,metadata_json,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
				id, in.Title, in.Description, "open", in.Priority, in.IssueType, "[]", "[]", "{}", now, now)
			if err == nil {
				return ShowTask(db, id)
			}
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				continue
			}
			return nil, err
		}
	}
	return nil, errors.New("failed to allocate unique id after retries")
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

func ReopenTask(db *sql.DB, id string) (*Task, error) {
	if _, err := ShowTask(db, id); err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`UPDATE tasks SET status='open',close_reason='',closed_at='',updated_at=? WHERE id=?`, now, id)
	if err != nil {
		return nil, err
	}
	return ShowTask(db, id)
}

func DeleteTask(db *sql.DB, id string) error {
	res, err := db.Exec(`DELETE FROM tasks WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("issue not found: %s", id)
	}
	return nil
}

func AddLabel(db *sql.DB, id, label string) (*Task, error) {
	if strings.TrimSpace(label) == "" {
		return nil, errors.New("label is required")
	}
	return UpdateTask(db, id, UpdateInput{AddLabels: []string{label}})
}

func RemoveLabel(db *sql.DB, id, label string) (*Task, error) {
	t, err := ShowTask(db, id)
	if err != nil {
		return nil, err
	}
	keep := make([]string, 0, len(t.Labels))
	for _, l := range t.Labels {
		if l != label {
			keep = append(keep, l)
		}
	}
	labelsRaw, _ := json.Marshal(keep)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.Exec(`UPDATE tasks SET labels_json=?,updated_at=? WHERE id=?`, string(labelsRaw), now, id)
	if err != nil {
		return nil, err
	}
	return ShowTask(db, id)
}

func ListLabels(db *sql.DB, id string) ([]string, error) {
	t, err := ShowTask(db, id)
	if err != nil {
		return nil, err
	}
	return t.Labels, nil
}

func ListAllLabels(db *sql.DB) ([]string, error) {
	tasks, err := ListTasks(db)
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, t := range tasks {
		for _, l := range t.Labels {
			set[l] = true
		}
	}
	out := make([]string, 0, len(set))
	for l := range set {
		out = append(out, l)
	}
	sort.Strings(out)
	return out, nil
}

func syncTaskDepsJSON(db *sql.DB, issueID string) error {
	rows, err := db.Query(`SELECT depends_on_id FROM dependencies WHERE issue_id=? ORDER BY depends_on_id`, issueID)
	if err != nil {
		return err
	}
	defer rows.Close()
	deps := make([]string, 0)
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return err
		}
		deps = append(deps, d)
	}
	raw, _ := json.Marshal(deps)
	_, err = db.Exec(`UPDATE tasks SET deps_json=?,updated_at=? WHERE id=?`, string(raw), time.Now().UTC().Format(time.RFC3339), issueID)
	return err
}

func AddDependency(db *sql.DB, issueID, dependsOnID, depType string) error {
	if depType == "" {
		depType = "blocks"
	}
	if _, err := ShowTask(db, issueID); err != nil {
		return err
	}
	if _, err := ShowTask(db, dependsOnID); err != nil {
		return err
	}
	_, err := db.Exec(`INSERT OR IGNORE INTO dependencies(issue_id,depends_on_id,dep_type) VALUES(?,?,?)`, issueID, dependsOnID, depType)
	if err != nil {
		return err
	}
	return syncTaskDepsJSON(db, issueID)
}

func RemoveDependency(db *sql.DB, issueID, dependsOnID string) error {
	res, err := db.Exec(`DELETE FROM dependencies WHERE issue_id=? AND depends_on_id=?`, issueID, dependsOnID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("dependency not found: %s -> %s", issueID, dependsOnID)
	}
	return syncTaskDepsJSON(db, issueID)
}

func ListDependencies(db *sql.DB, issueID string) ([]string, error) {
	rows, err := db.Query(`SELECT depends_on_id FROM dependencies WHERE issue_id=? ORDER BY depends_on_id`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func ListChildren(db *sql.DB, parentID string) ([]Task, error) {
	rows, err := db.Query(`SELECT issue_id FROM dependencies WHERE dep_type='parent-child' AND depends_on_id=? ORDER BY issue_id`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Task, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		t, err := ShowTask(db, id)
		if err != nil {
			continue
		}
		out = append(out, *t)
	}
	return out, nil
}

type BlockedItem struct {
	Task
	BlockedByCount int      `json:"blocked_by_count"`
	BlockedBy      []string `json:"blocked_by"`
}

func ListBlocked(db *sql.DB) ([]BlockedItem, error) {
	rows, err := db.Query(`
		SELECT d.depends_on_id, d.issue_id
		FROM dependencies d
		JOIN tasks blocker ON blocker.id = d.issue_id
		WHERE d.dep_type='blocks' AND blocker.status != 'closed'
		ORDER BY d.depends_on_id, d.issue_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byTarget := map[string][]string{}
	order := make([]string, 0)
	seen := map[string]bool{}
	for rows.Next() {
		var targetID, blockerID string
		if err := rows.Scan(&targetID, &blockerID); err != nil {
			return nil, err
		}
		byTarget[targetID] = append(byTarget[targetID], blockerID)
		if !seen[targetID] {
			seen[targetID] = true
			order = append(order, targetID)
		}
	}

	out := make([]BlockedItem, 0, len(order))
	for _, id := range order {
		t, err := ShowTask(db, id)
		if err != nil {
			continue
		}
		if t.Status == "closed" {
			continue
		}
		blockedBy := byTarget[id]
		out = append(out, BlockedItem{Task: *t, BlockedByCount: len(blockedBy), BlockedBy: blockedBy})
	}
	return out, nil
}

func ReadyTasks(db *sql.DB) ([]Task, error) {
	rows, err := db.Query(`SELECT DISTINCT issue_id FROM dependencies WHERE dep_type='blocks'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	blockedSelf := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		blockedSelf[id] = true
	}
	tasks, err := ListTasks(db)
	if err != nil {
		return nil, err
	}
	out := make([]Task, 0)
	for _, t := range tasks {
		if t.Status == "open" && !blockedSelf[t.ID] {
			out = append(out, t)
		}
	}
	return out, nil
}

func RenameTask(db *sql.DB, oldID, newID string) (*Task, error) {
	if strings.TrimSpace(oldID) == "" || strings.TrimSpace(newID) == "" {
		return nil, errors.New("old and new id are required")
	}
	if oldID == newID {
		return nil, errors.New("old and new id must differ")
	}
	if _, err := ShowTask(db, oldID); err != nil {
		return nil, err
	}
	if _, err := ShowTask(db, newID); err == nil {
		return nil, fmt.Errorf("issue already exists: %s", newID)
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	stmts := []struct {
		q    string
		args []any
	}{
		{`UPDATE tasks SET id=?,updated_at=? WHERE id=?`, []any{newID, time.Now().UTC().Format(time.RFC3339), oldID}},
		{`UPDATE dependencies SET issue_id=? WHERE issue_id=?`, []any{newID, oldID}},
		{`UPDATE dependencies SET depends_on_id=? WHERE depends_on_id=?`, []any{newID, oldID}},
		{`UPDATE comments SET issue_id=? WHERE issue_id=?`, []any{newID, oldID}},
	}
	for _, s := range stmts {
		if _, err := tx.Exec(s.q, s.args...); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// refresh deps_json for tasks potentially impacted by dependency id rewrites
	rows, err := db.Query(`SELECT DISTINCT issue_id FROM dependencies WHERE issue_id=? OR depends_on_id=?`, newID, newID)
	if err == nil {
		defer rows.Close()
		seen := map[string]bool{}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err == nil && !seen[id] {
				seen[id] = true
				_ = syncTaskDepsJSON(db, id)
			}
		}
	}

	return ShowTask(db, newID)
}

func RenamePrefix(db *sql.DB, oldPrefix, newPrefix string) (int, error) {
	if strings.TrimSpace(oldPrefix) == "" || strings.TrimSpace(newPrefix) == "" {
		return 0, errors.New("old and new prefix are required")
	}
	if oldPrefix == newPrefix {
		return 0, nil
	}
	tasks, err := ListTasks(db)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, t := range tasks {
		if strings.HasPrefix(t.ID, oldPrefix+"-") {
			newID := newPrefix + t.ID[len(oldPrefix):]
			if _, err := RenameTask(db, t.ID, newID); err != nil {
				return count, err
			}
			count++
		}
	}
	_, _ = db.Exec(`INSERT OR REPLACE INTO config(key,value) VALUES ('id_prefix', ?)`, newPrefix)
	return count, nil
}

type Comment struct {
	ID        int    `json:"id"`
	IssueID   string `json:"issue_id"`
	Author    string `json:"author,omitempty"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

func AddComment(db *sql.DB, issueID, author, body string) (*Comment, error) {
	if _, err := ShowTask(db, issueID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(body) == "" {
		return nil, errors.New("comment body is required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec(`INSERT INTO comments(issue_id,author,body,created_at) VALUES(?,?,?,?)`, issueID, author, body, now)
	if err != nil {
		return nil, err
	}
	id64, _ := res.LastInsertId()
	return &Comment{ID: int(id64), IssueID: issueID, Author: author, Body: body, CreatedAt: now}, nil
}

func ListComments(db *sql.DB, issueID string) ([]Comment, error) {
	if _, err := ShowTask(db, issueID); err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT id,issue_id,author,body,created_at FROM comments WHERE issue_id=? ORDER BY id ASC`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Comment, 0)
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.IssueID, &c.Author, &c.Body, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func SearchTasks(db *sql.DB, query string, limit int) ([]Task, error) {
	query = strings.TrimSpace(strings.ToLower(query))
	if limit <= 0 {
		limit = 50
	}
	tasks, err := ListTasks(db)
	if err != nil {
		return nil, err
	}
	out := make([]Task, 0)
	for _, t := range tasks {
		if query == "" || strings.Contains(strings.ToLower(t.ID), query) || strings.Contains(strings.ToLower(t.Title), query) || strings.Contains(strings.ToLower(t.Description), query) {
			out = append(out, t)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func CountTasks(db *sql.DB, statusFilter string) (int, error) {
	if strings.TrimSpace(statusFilter) == "" {
		row := db.QueryRow(`SELECT COUNT(*) FROM tasks`)
		var n int
		if err := row.Scan(&n); err != nil {
			return 0, err
		}
		return n, nil
	}
	row := db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE status=?`, statusFilter)
	var n int
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func StatusSummary(db *sql.DB) (map[string]int, error) {
	rows, err := db.Query(`SELECT status, COUNT(*) FROM tasks GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	summary := map[string]int{"open": 0, "in_progress": 0, "closed": 0, "resolved": 0}
	for rows.Next() {
		var s string
		var c int
		if err := rows.Scan(&s, &c); err != nil {
			return nil, err
		}
		summary[s] = c
	}
	return summary, nil
}

func StaleTasks(db *sql.DB, days int) ([]Task, error) {
	if days <= 0 {
		days = 30
	}
	tasks, err := ListTasks(db)
	if err != nil {
		return nil, err
	}
	threshold := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	out := make([]Task, 0)
	for _, t := range tasks {
		if t.Status != "open" {
			continue
		}
		ts := strings.TrimSpace(t.UpdatedAt)
		if ts == "" {
			ts = strings.TrimSpace(t.CreatedAt)
		}
		parsed, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			continue
		}
		if parsed.Before(threshold) {
			out = append(out, t)
		}
	}
	return out, nil
}

func OrphanTasks(db *sql.DB) ([]Task, error) {
	rows, err := db.Query(`
		SELECT DISTINCT d.issue_id
		FROM dependencies d
		LEFT JOIN tasks parent ON parent.id = d.depends_on_id
		WHERE parent.id IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Task, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		t, err := ShowTask(db, id)
		if err != nil {
			continue
		}
		out = append(out, *t)
	}
	return out, nil
}

func QueryTasks(db *sql.DB, expr string) ([]Task, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, errors.New("query expression required")
	}
	tasks, err := ListTasks(db)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(expr, "AND")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	out := make([]Task, 0)
	for _, t := range tasks {
		ok := true
		for _, p := range parts {
			if p == "" {
				continue
			}
			switch {
			case strings.Contains(p, ">="):
				kv := strings.SplitN(p, ">=", 2)
				if strings.TrimSpace(strings.ToLower(kv[0])) != "priority" {
					return nil, errors.New("only priority comparisons are supported")
				}
				n, _ := strconv.Atoi(strings.TrimSpace(kv[1]))
				ok = ok && t.Priority >= n
			case strings.Contains(p, "<="):
				kv := strings.SplitN(p, "<=", 2)
				if strings.TrimSpace(strings.ToLower(kv[0])) != "priority" {
					return nil, errors.New("only priority comparisons are supported")
				}
				n, _ := strconv.Atoi(strings.TrimSpace(kv[1]))
				ok = ok && t.Priority <= n
			case strings.Contains(p, ">"):
				kv := strings.SplitN(p, ">", 2)
				if strings.TrimSpace(strings.ToLower(kv[0])) != "priority" {
					return nil, errors.New("only priority comparisons are supported")
				}
				n, _ := strconv.Atoi(strings.TrimSpace(kv[1]))
				ok = ok && t.Priority > n
			case strings.Contains(p, "<"):
				kv := strings.SplitN(p, "<", 2)
				if strings.TrimSpace(strings.ToLower(kv[0])) != "priority" {
					return nil, errors.New("only priority comparisons are supported")
				}
				n, _ := strconv.Atoi(strings.TrimSpace(kv[1]))
				ok = ok && t.Priority < n
			case strings.Contains(p, "="):
				kv := strings.SplitN(p, "=", 2)
				k := strings.TrimSpace(strings.ToLower(kv[0]))
				v := strings.TrimSpace(kv[1])
				switch k {
				case "status":
					ok = ok && strings.EqualFold(t.Status, v)
				case "type":
					ok = ok && strings.EqualFold(t.IssueType, v)
				case "assignee":
					ok = ok && strings.EqualFold(t.Assignee, v)
				case "title":
					ok = ok && strings.Contains(strings.ToLower(t.Title), strings.ToLower(v))
				default:
					return nil, fmt.Errorf("unknown field: %s", k)
				}
			default:
				return nil, fmt.Errorf("invalid query clause: %s", p)
			}
			if !ok {
				break
			}
		}
		if ok {
			out = append(out, t)
		}
	}
	return out, nil
}
