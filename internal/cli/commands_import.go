package cli

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	if !hasTable(src, "tasks") {
		return errors.New("source validation failed: required table 'tasks' missing")
	}
	return nil
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

	rows, err := src.Query(`SELECT id,title,COALESCE(description,''),COALESCE(status,'open'),COALESCE(priority,2),COALESCE(issue_type,'task'),COALESCE(assignee,''),COALESCE(owner,''),COALESCE(labels_json,'[]'),COALESCE(deps_json,'[]'),COALESCE(metadata_json,'{}'),COALESCE(close_reason,''),COALESCE(created_at,''),COALESCE(updated_at,''),COALESCE(closed_at,'') FROM tasks ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("read source tasks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, title, desc, status, issueType, assignee, owner, labels, deps, metadata, closeReason, createdAt, updatedAt, closedAt string
		var priority int
		if err := rows.Scan(&id, &title, &desc, &status, &priority, &issueType, &assignee, &owner, &labels, &deps, &metadata, &closeReason, &createdAt, &updatedAt, &closedAt); err != nil {
			return nil, fmt.Errorf("mapping tasks row: %w", err)
		}
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
				id, title, desc, status, priority, issueType, assignee, owner, labels, deps, metadata, closeReason, createdAt, updatedAt, closedAt)
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate source tasks: %w", err)
	}

	if hasTable(src, "dependencies") {
		depRows, err := src.Query(`SELECT issue_id,depends_on_id,COALESCE(dep_type,'blocks') FROM dependencies`)
		if err != nil {
			return nil, fmt.Errorf("read source dependencies: %w", err)
		}
		defer depRows.Close()
		for depRows.Next() {
			var issueID, dependsOnID, depType string
			if err := depRows.Scan(&issueID, &dependsOnID, &depType); err != nil {
				return nil, fmt.Errorf("mapping dependencies row: %w", err)
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
