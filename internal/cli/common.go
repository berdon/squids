package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/berdon/squids/internal/store"
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

func printGlobalFlags() {
	fmt.Println("Global Flags:")
	fmt.Println("      --actor string              Actor name for audit trail (default: $BD_ACTOR, git user.name, $USER)")
	fmt.Println("      --db string                 Database path (default: auto-discover .sq/*.db)")
	fmt.Println("      --dolt-auto-commit string   Compatibility flag accepted for bd parity (no-op on sqlite backend)")
	fmt.Println("      --json                      Output in JSON format")
	fmt.Println("      --profile                   Generate CPU profile for performance analysis")
	fmt.Println("  -q, --quiet                     Suppress non-essential output (errors only)")
	fmt.Println("      --readonly                  Read-only mode: block write operations (for worker sandboxes)")
	fmt.Println("      --sandbox                   Sandbox mode: disables auto-sync")
	fmt.Println("  -v, --verbose                   Enable verbose/debug output")
}

func usage() {
	fmt.Println("sq - squids task CLI")
	fmt.Println("Issues chained together like squids. A lightweight issue tracker with first-class dependency support.")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  sq [flags]")
	fmt.Println("  sq [command]")
	fmt.Println("")
	fmt.Println("Working With Issues:")
	fmt.Println("  children      List child tasks for a parent")
	fmt.Println("  close         Close one or more issues")
	fmt.Println("  comments      View or manage comments on an issue")
	fmt.Println("  create        Create a task")
	fmt.Println("  delete        Delete one or more issues")
	fmt.Println("  edit          Edit issue in configured editor (compat surface)")
	fmt.Println("  gate          Manage workflow gates (compat surface)")
	fmt.Println("  label         Manage issue labels")
	fmt.Println("  list          List tasks")
	fmt.Println("  q             Quick-capture task and output ID")
	fmt.Println("  query         Query tasks")
	fmt.Println("  reopen        Reopen one or more closed issues")
	fmt.Println("  search        Search tasks")
	fmt.Println("  set-state     Set operational state label (compat surface)")
	fmt.Println("  show          Show a task")
	fmt.Println("  todo          Manage TODO items")
	fmt.Println("  update        Update a task")
	fmt.Println("")
	fmt.Println("Views & Reports:")
	fmt.Println("  count         Count tasks")
	fmt.Println("  history       Show issue history (not supported on sqlite backend)")
	fmt.Println("  stale         List stale open tasks")
	fmt.Println("  status        Show status summary")
	fmt.Println("  types         List supported issue types")
	fmt.Println("")
	fmt.Println("Dependencies & Structure:")
	fmt.Println("  blocked       Show blocked tasks")
	fmt.Println("  dep           Manage dependencies")
	fmt.Println("  duplicate     Mark issue as duplicate of canonical issue")
	fmt.Println("  orphans       List tasks with orphaned dependency refs")
	fmt.Println("  supersede     Mark issue as superseded by replacement")
	fmt.Println("  swarm         Swarm management commands (compat surface)")
	fmt.Println("")
	fmt.Println("Sync & Data:")
	fmt.Println("  backup        Backup/restore sq sqlite database (compat surface)")
	fmt.Println("  import-beads  Import tasks/deps/comments from a beads sqlite DB")
	fmt.Println("  restore       Compatibility command (sqlite backend has no Dolt restore)")
	fmt.Println("")
	fmt.Println("Setup & Configuration:")
	fmt.Println("  dolt          Dolt integration commands (compat surface)")
	fmt.Println("  hooks         Manage git hooks (compat surface)")
	fmt.Println("  human         Human-focused command group")
	fmt.Println("  info          Show database information")
	fmt.Println("  init          Initialize task storage")
	fmt.Println("  memories      Persistent memory store commands (compat surface)")
	fmt.Println("  onboard       Print AGENTS.md onboarding snippet")
	fmt.Println("  quickstart    Show quick start guide")
	fmt.Println("  ready         Check backend readiness")
	fmt.Println("  setup         Setup editor/assistant integration files (compat surface)")
	fmt.Println("  where         Show active sq storage location")
	fmt.Println("")
	fmt.Println("Additional Commands:")
	fmt.Println("  audit         Audit interaction log commands (compat surface)")
	fmt.Println("  completion    Generate shell completion scripts")
	fmt.Println("  defer         Defer one or more tasks")
	fmt.Println("  gitlab        GitLab integration commands (compat surface)")
	fmt.Println("  help          Help about any command")
	fmt.Println("  linear        Linear integration commands (compat surface)")
	fmt.Println("  mail          Mail provider delegation (compat surface)")
	fmt.Println("  mol           Molecule/work-template commands (compat surface)")
	fmt.Println("  purge         Purge closed ephemeral tasks (compat surface)")
	fmt.Println("  rename        Rename an issue ID")
	fmt.Println("  rename-prefix Rename issue ID prefix")
	fmt.Println("  undefer       Restore deferred tasks to open")
	fmt.Println("  version       Show CLI version")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  -h, --help     help for sq")
	fmt.Println("  -V, --version  print version information")
	fmt.Println("")
	printGlobalFlags()
	fmt.Println("")
	fmt.Println("Use \"sq [command] --help\" for more information about a command.")
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
