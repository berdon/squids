package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gitea/auhanson/squids/internal/store"
)

// Package cli owns command parsing/dispatch for the sq binary.
//
// Store and data-layer concerns live in internal/store; this package is
// intentionally focused on translating CLI input into store operations.

func printJSON(v any) int {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode json: %v\n", err)
		return 1
	}
	return 0
}

func failUsage(msg string) int {
	fmt.Fprintln(os.Stderr, msg)
	return 2
}

func failRuntime(msg string) int {
	fmt.Fprintln(os.Stderr, msg)
	return 1
}

func usage() {
	fmt.Println("sq - squids task CLI")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  sq <command> [args]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  help    Show help for commands")
	fmt.Println("  init    Initialize task storage")
	fmt.Println("  ready   Check backend readiness")
	fmt.Println("  create  Create a task")
	fmt.Println("  q       Quick-create a task and output ID")
	fmt.Println("  show    Show a task")
	fmt.Println("  list    List tasks")
	fmt.Println("  update  Update a task")
	fmt.Println("  close   Close a task")
	fmt.Println("  reopen  Reopen a task")
	fmt.Println("  delete  Delete a task")
	fmt.Println("  label   Manage labels")
	fmt.Println("  dep     Manage dependencies")
	fmt.Println("  comments Manage comments")
	fmt.Println("  todo    Manage TODO items")
	fmt.Println("  children List child tasks for a parent")
	fmt.Println("  blocked Show blocked tasks")
	fmt.Println("  defer   Defer one or more tasks")
	fmt.Println("  undefer Restore deferred tasks to open")
	fmt.Println("  rename  Rename an issue ID")
	fmt.Println("  rename-prefix Rename issue ID prefix")
	fmt.Println("  duplicate Mark issue as duplicate of canonical issue")
	fmt.Println("  supersede Mark issue as superseded by replacement")
	fmt.Println("  types   List supported issue types")
	fmt.Println("  query   Query tasks")
	fmt.Println("  stale   List stale open tasks")
	fmt.Println("  orphans List tasks with orphaned dependency refs")
	fmt.Println("  search  Search tasks")
	fmt.Println("  count   Count tasks")
	fmt.Println("  status  Show status summary")
	fmt.Println("  version Show CLI version")
	fmt.Println("  where   Show active sq storage location")
	fmt.Println("  info    Show database information")
	fmt.Println("  human   Human-focused command group")
}

func dbPathFromEnvOrCwd() (string, error) {
	if explicit := os.Getenv("SQ_DB_PATH"); explicit != "" {
		return explicit, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve cwd: %w", err)
	}
	return store.DefaultDBPath(filepath.Clean(cwd)), nil
}

func openTaskDB() (*sql.DB, string, error) {
	dbPath, err := dbPathFromEnvOrCwd()
	if err != nil {
		return nil, "", err
	}
	db, err := store.Open(dbPath)
	if err != nil {
		return nil, dbPath, err
	}
	if err := store.EnsureInitialized(db); err != nil {
		_ = db.Close()
		return nil, dbPath, err
	}
	return db, dbPath, nil
}
