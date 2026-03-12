package cli

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type importOptions struct {
	Source     string
	DryRun     bool
	JSON       bool
	NoComments bool
	NoEvents   bool
}

type importReport struct {
	Source    string         `json:"source"`
	DryRun    bool           `json:"dry_run"`
	Tasks     map[string]int `json:"tasks"`
	Deps      map[string]int `json:"deps"`
	Comments  map[string]int `json:"comments"`
	Warnings  []string       `json:"warnings,omitempty"`
	ElapsedMS int64          `json:"elapsed_ms"`
}

func cmdEdit(args []string) int {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--help", "-h", "--json", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags
		case "--actor", "--db", "--dolt-auto-commit":
			if i+1 < len(args) {
				i++
			}
		default:
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
		}
	}
	return failRuntime("edit command requires editor integration; sq does not implement editor workflows yet")
}

func cmdDolt(args []string) int {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--help", "-h", "--json", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags
		case "--actor", "--db", "--dolt-auto-commit":
			if i+1 < len(args) {
				i++
			}
		default:
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
		}
	}
	return failRuntime("dolt integration not yet supported on sq sqlite backend")
}

func cmdLinear(args []string) int {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--help", "-h", "--json", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags
		case "--actor", "--db", "--dolt-auto-commit":
			if i+1 < len(args) {
				i++
			}
		default:
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
		}
	}
	return failRuntime("linear integration not yet supported on sq sqlite backend")
}

func cmdMemories(args []string) int {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--help", "-h", "--json", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags
		case "--actor", "--db", "--dolt-auto-commit":
			if i+1 < len(args) {
				i++
			}
		default:
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
		}
	}
	return failRuntime("memories compatibility surface only; sq has no persistent memory store yet")
}

func cmdGitLab(args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "sq gitlab [projects|status|sync]")
		return 0
	}
	sub := args[0]
	for i := 1; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--help", "-h", "--json", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags
		case "--actor", "--db", "--dolt-auto-commit":
			if i+1 < len(args) {
				i++
			}
		default:
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
		}
	}
	switch sub {
	case "projects", "status", "sync":
		return failRuntime("gitlab integration not yet supported on sq sqlite backend")
	default:
		return failUsage("unknown gitlab subcommand: " + sub)
	}
}

func cmdImportBeads(args []string) int {
	opts, err := parseImportOptions(args)
	if err != nil {
		return failUsage(err.Error())
	}

	sourcePath, err := resolveImportSource(opts.Source)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	srcDB, err := openSQLite(sourcePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open source db: %v\n", err)
		return 2
	}
	defer srcDB.Close()

	if err := validateSourceSchema(srcDB); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	tgtDB, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer tgtDB.Close()

	started := time.Now()
	report, err := importFromSource(srcDB, tgtDB, sourcePath, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		if strings.Contains(strings.ToLower(err.Error()), "validate") || strings.Contains(strings.ToLower(err.Error()), "mapping") {
			return 3
		}
		return 4
	}
	report.ElapsedMS = time.Since(started).Milliseconds()

	if opts.JSON {
		return printJSON(report)
	}

	fmt.Printf("Imported from: %s\n", report.Source)
	fmt.Printf("Tasks: created=%d updated=%d unchanged=%d\n", report.Tasks["created"], report.Tasks["updated"], report.Tasks["unchanged"])
	fmt.Printf("Deps: created=%d unchanged=%d\n", report.Deps["created"], report.Deps["unchanged"])
	if !opts.NoComments {
		fmt.Printf("Comments: created=%d unchanged=%d\n", report.Comments["created"], report.Comments["unchanged"])
	}
	fmt.Printf("Dry-run: %t\n", report.DryRun)
	return 0
}

func parseImportOptions(args []string) (importOptions, error) {
	opts := importOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--source":
			if i+1 >= len(args) {
				return opts, errors.New("usage: sq import-beads [--source <path>] [--dry-run] [--json] [--no-comments] [--no-events]")
			}
			opts.Source = args[i+1]
			i++
		case "--dry-run":
			opts.DryRun = true
		case "--json":
			opts.JSON = true
		case "--no-comments":
			opts.NoComments = true
		case "--no-events":
			opts.NoEvents = true
		default:
			return opts, fmt.Errorf("unknown flag: %s", args[i])
		}
	}
	return opts, nil
}

func openSQLite(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_busy_timeout=5000&_foreign_keys=1", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func resolveImportSource(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return normalizeSourcePath(explicit)
	}

	candidates := make([]string, 0)
	if db := strings.TrimSpace(os.Getenv("BEADS_DATABASE")); db != "" {
		if p, err := normalizeSourcePath(db); err == nil {
			candidates = append(candidates, p)
		}
	}
	if dir := strings.TrimSpace(os.Getenv("BEADS_DIR")); dir != "" {
		if p, err := normalizeSourcePath(dir); err == nil {
			candidates = append(candidates, p)
		}
	}

	cwd, err := os.Getwd()
	if err == nil {
		for dir := cwd; ; dir = filepath.Dir(dir) {
			beadsDir := filepath.Join(dir, ".beads")
			if p, err := normalizeSourcePath(beadsDir); err == nil {
				candidates = append(candidates, p)
			}
			if parent := filepath.Dir(dir); parent == dir {
				break
			}
		}
	}

	uniq := map[string]bool{}
	filtered := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if !uniq[c] {
			uniq[c] = true
			filtered = append(filtered, c)
		}
	}

	if len(filtered) == 0 {
		return "", errors.New("unable to discover beads source database; pass --source <path>")
	}
	if len(filtered) > 1 {
		return "", fmt.Errorf("multiple beads source candidates found (%s); pass --source <path>", strings.Join(filtered, ", "))
	}
	return filtered[0], nil
}

func normalizeSourcePath(input string) (string, error) {
	p := filepath.Clean(input)
	st, err := os.Stat(p)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return p, nil
	}

	known := []string{"tasks.sqlite", "beads.sqlite", "db.sqlite", "beads.db", "bd.db"}
	for _, name := range known {
		cand := filepath.Join(p, name)
		if _, err := os.Stat(cand); err == nil {
			return cand, nil
		}
	}

	entries, err := os.ReadDir(p)
	if err != nil {
		return "", err
	}
	matches := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.ToLower(e.Name())
		if strings.HasSuffix(name, ".db") || strings.HasSuffix(name, ".sqlite") {
			matches = append(matches, filepath.Join(p, e.Name()))
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("ambiguous sqlite files in %s", p)
	}
	return "", fmt.Errorf("no sqlite file found in %s", p)
}

func hasTable(db *sql.DB, name string) bool {
	row := db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name=?`, name)
	var n int
	if err := row.Scan(&n); err != nil {
		return false
	}
	return n > 0
}

func validateSourceSchema(src *sql.DB) error {
	if hasTable(src, "tasks") || hasTable(src, "issues") {
		return nil
	}
	return errors.New("source validation failed: required table 'tasks' or 'issues' missing")
}

func importFromSource(src, dst *sql.DB, sourcePath string, opts importOptions) (*importReport, error) {
	report := &importReport{
		Source: sourcePath,
		DryRun: opts.DryRun,
		Tasks: map[string]int{"created": 0, "updated": 0, "unchanged": 0},
		Deps: map[string]int{"created": 0, "unchanged": 0},
		Comments: map[string]int{"created": 0, "unchanged": 0},
	}

	tx, err := dst.Begin()
	if err != nil {
		return nil, fmt.Errorf("write begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	tasks, schemaVariant, err := loadSourceTasks(src)
	if err != nil {
		return nil, err
	}

	for _, task := range tasks {
		id, title := task["id"], task["title"]
		if strings.TrimSpace(id) == "" || strings.TrimSpace(title) == "" {
			report.Warnings = append(report.Warnings, "skipped task with empty id/title")
			continue
		}

		var existing int
		if err := tx.QueryRow(`SELECT COUNT(1) FROM tasks WHERE id=?`, id).Scan(&existing); err != nil {
			return nil, fmt.Errorf("check existing task %s: %w", id, err)
		}

		if !opts.DryRun {
			_, err = tx.Exec(`
INSERT INTO tasks(id,title,description,status,priority,issue_type,assignee,owner,labels_json,deps_json,metadata_json,close_reason,created_at,updated_at,closed_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  title=excluded.title,
  description=excluded.description,
  status=excluded.status,
  priority=excluded.priority,
  issue_type=excluded.issue_type,
  assignee=excluded.assignee,
  owner=excluded.owner,
  labels_json=excluded.labels_json,
  deps_json=excluded.deps_json,
  metadata_json=excluded.metadata_json,
  close_reason=excluded.close_reason,
  created_at=excluded.created_at,
  updated_at=excluded.updated_at,
  closed_at=excluded.closed_at`,
				id,
				title,
				task["description"],
				task["status"],
				toInt(task["priority"], 2),
				task["issue_type"],
				task["assignee"],
				task["owner"],
				task["labels_json"],
				task["deps_json"],
				task["metadata_json"],
				task["close_reason"],
				task["created_at"],
				task["updated_at"],
				task["closed_at"],
			)
			if err != nil {
				return nil, fmt.Errorf("write task %s: %w", id, err)
			}
		}

		if existing == 0 {
			report.Tasks["created"]++
		} else {
			report.Tasks["updated"]++
		}
	}

	if schemaVariant == "issues" {
		report.Warnings = append(report.Warnings, "imported from issues-style beads schema")
	}

	if hasTable(src, "dependencies") || hasTable(src, "issue_deps") {
		depQuery := `SELECT issue_id,depends_on_id,COALESCE(dep_type,'blocks') FROM dependencies`
		depName := "dependencies"
		if hasTable(src, "issue_deps") && !hasTable(src, "dependencies") {
			depQuery = `SELECT issue_id,depends_on_id,COALESCE(dep_type,'blocks') FROM issue_deps`
			depName = "issue_deps"
		}
		depRows, err := src.Query(depQuery)
		if err != nil {
			return nil, fmt.Errorf("read source %s: %w", depName, err)
		}
		defer depRows.Close()
		for depRows.Next() {
			var issueID, dependsOnID, depType string
			if err := depRows.Scan(&issueID, &dependsOnID, &depType); err != nil {
				return nil, fmt.Errorf("mapping %s row: %w", depName, err)
			}
			if issueID == dependsOnID || issueID == "" || dependsOnID == "" {
				report.Warnings = append(report.Warnings, fmt.Sprintf("skipped invalid dependency %s->%s", issueID, dependsOnID))
				continue
			}
			var existing int
			if err := tx.QueryRow(`SELECT COUNT(1) FROM dependencies WHERE issue_id=? AND depends_on_id=? AND dep_type=?`, issueID, dependsOnID, depType).Scan(&existing); err != nil {
				return nil, err
			}
			if !opts.DryRun {
				if _, err := tx.Exec(`INSERT OR IGNORE INTO dependencies(issue_id,depends_on_id,dep_type) VALUES(?,?,?)`, issueID, dependsOnID, depType); err != nil {
					return nil, fmt.Errorf("write dependency %s->%s: %w", issueID, dependsOnID, err)
				}
			}
			if existing == 0 {
				report.Deps["created"]++
			} else {
				report.Deps["unchanged"]++
			}
		}
	}

	if !opts.NoComments && hasTable(src, "comments") {
		commentRows, err := src.Query(`SELECT issue_id,COALESCE(author,''),body,COALESCE(created_at,'') FROM comments ORDER BY created_at, id`)
		if err != nil {
			return nil, fmt.Errorf("read source comments: %w", err)
		}
		defer commentRows.Close()
		for commentRows.Next() {
			var issueID, author, body, createdAt string
			if err := commentRows.Scan(&issueID, &author, &body, &createdAt); err != nil {
				return nil, fmt.Errorf("mapping comments row: %w", err)
			}
			if strings.TrimSpace(issueID) == "" || strings.TrimSpace(body) == "" {
				continue
			}
			var existing int
			if err := tx.QueryRow(`SELECT COUNT(1) FROM comments WHERE issue_id=? AND author=? AND body=? AND created_at=?`, issueID, author, body, createdAt).Scan(&existing); err != nil {
				return nil, err
			}
			if !opts.DryRun {
				if _, err := tx.Exec(`INSERT INTO comments(issue_id,author,body,created_at) SELECT ?,?,?,? WHERE NOT EXISTS (SELECT 1 FROM comments WHERE issue_id=? AND author=? AND body=? AND created_at=?)`, issueID, author, body, createdAt, issueID, author, body, createdAt); err != nil {
					return nil, fmt.Errorf("write comment issue=%s: %w", issueID, err)
				}
			}
			if existing == 0 {
				report.Comments["created"]++
			} else {
				report.Comments["unchanged"]++
			}
		}
	}

	if opts.DryRun {
		return report, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit import: %w", err)
	}
	return report, nil
}

func toInt(raw string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return def
	}
	return n
}

func loadSourceTasks(src *sql.DB) ([]map[string]string, string, error) {
	if hasTable(src, "tasks") {
		rows, err := src.Query(`SELECT id,title,COALESCE(description,''),COALESCE(status,'open'),COALESCE(priority,2),COALESCE(issue_type,'task'),COALESCE(assignee,''),COALESCE(owner,''),COALESCE(labels_json,'[]'),COALESCE(deps_json,'[]'),COALESCE(metadata_json,'{}'),COALESCE(close_reason,''),COALESCE(created_at,''),COALESCE(updated_at,''),COALESCE(closed_at,'') FROM tasks ORDER BY created_at, id`)
		if err != nil {
			return nil, "", fmt.Errorf("read source tasks: %w", err)
		}
		defer rows.Close()
		out := make([]map[string]string, 0)
		for rows.Next() {
			var id, title, desc, status, issueType, assignee, owner, labels, deps, metadata, closeReason, createdAt, updatedAt, closedAt string
			var priority int
			if err := rows.Scan(&id, &title, &desc, &status, &priority, &issueType, &assignee, &owner, &labels, &deps, &metadata, &closeReason, &createdAt, &updatedAt, &closedAt); err != nil {
				return nil, "", fmt.Errorf("mapping tasks row: %w", err)
			}
			out = append(out, map[string]string{
				"id": id, "title": title, "description": desc, "status": status, "priority": strconv.Itoa(priority), "issue_type": issueType,
				"assignee": assignee, "owner": owner, "labels_json": labels, "deps_json": deps, "metadata_json": metadata,
				"close_reason": closeReason, "created_at": createdAt, "updated_at": updatedAt, "closed_at": closedAt,
			})
		}
		if err := rows.Err(); err != nil {
			return nil, "", fmt.Errorf("iterate source tasks: %w", err)
		}
		return out, "tasks", nil
	}

	if !hasTable(src, "issues") {
		return nil, "", errors.New("source validation failed: required table 'tasks' or 'issues' missing")
	}

	rows, err := src.Query(`SELECT id,COALESCE(title,''),COALESCE(description,''),COALESCE(status,'open'),CAST(COALESCE(priority,2) AS TEXT),COALESCE(issue_type,'task'),COALESCE(assignee,''),COALESCE(owner,''),COALESCE(metadata_json,'{}'),COALESCE(close_reason,''),COALESCE(created_at,''),COALESCE(updated_at,''),COALESCE(closed_at,'') FROM issues ORDER BY created_at, id`)
	if err != nil {
		return nil, "", fmt.Errorf("read source issues: %w", err)
	}
	defer rows.Close()

	out := make([]map[string]string, 0)
	for rows.Next() {
		var id, title, desc, status, priority, issueType, assignee, owner, metadata, closeReason, createdAt, updatedAt, closedAt string
		if err := rows.Scan(&id, &title, &desc, &status, &priority, &issueType, &assignee, &owner, &metadata, &closeReason, &createdAt, &updatedAt, &closedAt); err != nil {
			return nil, "", fmt.Errorf("mapping issues row: %w", err)
		}
		labelsJSON, err := loadIssueLabelsJSON(src, id)
		if err != nil {
			return nil, "", err
		}
		depsJSON, err := loadIssueDepsJSON(src, id)
		if err != nil {
			return nil, "", err
		}
		out = append(out, map[string]string{
			"id": id, "title": title, "description": desc, "status": status, "priority": priority, "issue_type": issueType,
			"assignee": assignee, "owner": owner, "labels_json": labelsJSON, "deps_json": depsJSON, "metadata_json": metadata,
			"close_reason": closeReason, "created_at": createdAt, "updated_at": updatedAt, "closed_at": closedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate source issues: %w", err)
	}
	return out, "issues", nil
}

func loadIssueLabelsJSON(src *sql.DB, issueID string) (string, error) {
	if !hasTable(src, "issue_labels") {
		return "[]", nil
	}
	rows, err := src.Query(`SELECT label FROM issue_labels WHERE issue_id=? ORDER BY label`, issueID)
	if err != nil {
		return "", fmt.Errorf("read issue labels for %s: %w", issueID, err)
	}
	defer rows.Close()
	labels := make([]string, 0)
	for rows.Next() {
		var label string
		if err := rows.Scan(&label); err != nil {
			return "", fmt.Errorf("mapping issue_labels row: %w", err)
		}
		if strings.TrimSpace(label) != "" {
			labels = append(labels, label)
		}
	}
	b, _ := json.Marshal(labels)
	return string(b), nil
}

func loadIssueDepsJSON(src *sql.DB, issueID string) (string, error) {
	if !hasTable(src, "issue_deps") {
		return "[]", nil
	}
	rows, err := src.Query(`SELECT depends_on_id FROM issue_deps WHERE issue_id=? ORDER BY depends_on_id`, issueID)
	if err != nil {
		return "", fmt.Errorf("read issue deps for %s: %w", issueID, err)
	}
	defer rows.Close()
	deps := make([]string, 0)
	for rows.Next() {
		var dep string
		if err := rows.Scan(&dep); err != nil {
			return "", fmt.Errorf("mapping issue_deps row: %w", err)
		}
		if strings.TrimSpace(dep) != "" {
			deps = append(deps, dep)
		}
	}
	b, _ := json.Marshal(deps)
	return string(b), nil
}
