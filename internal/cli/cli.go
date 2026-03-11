package cli

// Package cli owns command parsing/dispatch for the sq binary.
//
// Store and data-layer concerns live in internal/store; this package is
// intentionally focused on translating CLI input into store operations.

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gitea/auhanson/squids/internal/store"
)

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
	fmt.Println("  init    Initialize task storage")
	fmt.Println("  ready   Check backend readiness")
	fmt.Println("  create  Create a task")
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
	fmt.Println("  duplicate Mark issue as duplicate of canonical issue")
	fmt.Println("  supersede Mark issue as superseded by replacement")
	fmt.Println("  types   List supported issue types")
	fmt.Println("  query   Query tasks")
	fmt.Println("  search  Search tasks")
	fmt.Println("  count   Count tasks")
	fmt.Println("  status  Show status summary")
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

func cmdInit() int {
	dbPath, err := dbPathFromEnvOrCwd()
	if err != nil {
		return failRuntime(err.Error())
	}
	db, err := store.Open(dbPath)
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	if err := store.Init(db); err != nil {
		return failRuntime(err.Error())
	}
	v, err := store.CurrentVersion(db)
	if err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(map[string]any{"command": "init", "ok": true, "db_path": dbPath, "schema_version": v})
}

func cmdReady() int {
	dbPath, err := dbPathFromEnvOrCwd()
	if err != nil {
		return failRuntime(err.Error())
	}
	db, err := store.Open(dbPath)
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	if err := store.EnsureInitialized(db); err != nil {
		return failRuntime(err.Error())
	}
	if err := store.Ping(db); err != nil {
		return failRuntime(err.Error())
	}
	v, err := store.CurrentVersion(db)
	if err != nil {
		return failRuntime(err.Error())
	}
	return printJSON([]map[string]any{{"command": "ready", "ok": true, "db_path": dbPath, "schema_version": v}})
}

func cmdCreate(args []string) int {
	if len(args) == 0 {
		return failUsage("title is required")
	}
	creator := strings.TrimSpace(os.Getenv("BD_ACTOR"))
	if creator == "" {
		creator = strings.TrimSpace(os.Getenv("USER"))
	}
	in := store.CreateInput{Title: args[0], IssueType: "task", Creator: creator}
	deps := make([][2]string, 0)
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 < len(args) {
				in.IssueType = args[i+1]
				i++
			}
		case "--priority":
			if i+1 < len(args) {
				p, err := strconv.Atoi(args[i+1])
				if err != nil {
					return failUsage("invalid --priority")
				}
				in.Priority = p
				i++
			}
		case "--description":
			if i+1 < len(args) {
				in.Description = args[i+1]
				i++
			}
		case "--deps":
			if i+1 < len(args) {
				spec := args[i+1]
				depType := "blocks"
				depID := spec
				if strings.Contains(spec, ":") {
					parts := strings.SplitN(spec, ":", 2)
					depType = strings.TrimSpace(parts[0])
					depID = strings.TrimSpace(parts[1])
				}
				if depID == "" {
					return failUsage("invalid --deps value")
				}
				deps = append(deps, [2]string{depType, depID})
				i++
			}
		case "--json":
			// accepted, no-op
		default:
			if strings.HasPrefix(args[i], "-") {
				return failUsage("unknown flag: " + args[i])
			}
		}
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	t, err := store.CreateTask(db, in)
	if err != nil {
		return failRuntime(err.Error())
	}
	for _, d := range deps {
		if err := store.AddDependency(db, t.ID, d[1], d[0]); err != nil {
			return failRuntime(err.Error())
		}
	}
	t, err = store.ShowTask(db, t.ID)
	if err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(t)
}

func cmdShow(args []string) int {
	if len(args) == 0 {
		return failUsage("id is required")
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	t, err := store.ShowTask(db, args[0])
	if err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(t)
}

func cmdList(args []string) int {
	for _, a := range args {
		if a == "--json" || a == "--flat" || a == "--no-pager" {
			continue
		}
		if strings.HasPrefix(a, "-") {
			return failUsage("unknown flag: " + a)
		}
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	tasks, err := store.ListTasks(db)
	if err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(tasks)
}

func cmdUpdate(args []string) int {
	if len(args) == 0 {
		return failUsage("id is required")
	}
	id := args[0]
	in := store.UpdateInput{SetMetadata: map[string]string{}}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--status":
			if i+1 < len(args) {
				v := args[i+1]
				in.Status = &v
				i++
			}
		case "--assignee":
			if i+1 < len(args) {
				v := args[i+1]
				in.Assignee = &v
				i++
			}
		case "--add-label":
			if i+1 < len(args) {
				in.AddLabels = append(in.AddLabels, args[i+1])
				i++
			}
		case "--set-metadata":
			if i+1 < len(args) {
				kv := strings.SplitN(args[i+1], "=", 2)
				if len(kv) != 2 {
					return failUsage("invalid --set-metadata value")
				}
				in.SetMetadata[kv[0]] = kv[1]
				i++
			}
		case "--claim":
			in.Claim = true
		case "--json":
			// accepted, no-op
		default:
			if strings.HasPrefix(args[i], "-") {
				return failUsage("unknown flag: " + args[i])
			}
		}
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	t, err := store.UpdateTask(db, id, in)
	if err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(t)
}

func cmdClose(args []string) int {
	if len(args) == 0 {
		return failUsage("id is required")
	}
	id := args[0]
	reason := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--reason":
			if i+1 < len(args) {
				reason = args[i+1]
				i++
			}
		case "--json":
			// accepted, no-op
		default:
			if strings.HasPrefix(args[i], "-") {
				return failUsage("unknown flag: " + args[i])
			}
		}
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	t, err := store.CloseTask(db, id, reason)
	if err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(t)
}

func cmdReopen(args []string) int {
	if len(args) == 0 {
		return failUsage("id is required")
	}
	id := args[0]
	for i := 1; i < len(args); i++ {
		if args[i] == "--json" {
			continue
		}
		if strings.HasPrefix(args[i], "-") {
			return failUsage("unknown flag: " + args[i])
		}
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	t, err := store.ReopenTask(db, id)
	if err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(t)
}

func cmdDelete(args []string) int {
	if len(args) == 0 {
		return failUsage("id is required")
	}
	id := args[0]
	for i := 1; i < len(args); i++ {
		if args[i] == "--json" || args[i] == "--force" {
			continue
		}
		if strings.HasPrefix(args[i], "-") {
			return failUsage("unknown flag: " + args[i])
		}
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	if err := store.DeleteTask(db, id); err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(map[string]any{"id": id, "deleted": true})
}

func cmdLabel(args []string) int {
	if len(args) == 0 {
		return failUsage("label subcommand required")
	}
	sub := args[0]
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()

	switch sub {
	case "add":
		if len(args) < 3 {
			return failUsage("usage: sq label add <id> <label> [--json]")
		}
		t, err := store.AddLabel(db, args[1], args[2])
		if err != nil {
			return failRuntime(err.Error())
		}
		return printJSON(t)
	case "remove":
		if len(args) < 3 {
			return failUsage("usage: sq label remove <id> <label> [--json]")
		}
		t, err := store.RemoveLabel(db, args[1], args[2])
		if err != nil {
			return failRuntime(err.Error())
		}
		return printJSON(t)
	case "list":
		if len(args) < 2 {
			return failUsage("usage: sq label list <id> [--json]")
		}
		labels, err := store.ListLabels(db, args[1])
		if err != nil {
			return failRuntime(err.Error())
		}
		return printJSON(labels)
	case "list-all":
		labels, err := store.ListAllLabels(db)
		if err != nil {
			return failRuntime(err.Error())
		}
		return printJSON(labels)
	default:
		return failUsage("unknown label subcommand: " + sub)
	}
}

func cmdDep(args []string) int {
	if len(args) == 0 {
		return failUsage("dep subcommand required")
	}
	sub := args[0]
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()

	switch sub {
	case "add":
		if len(args) < 3 {
			return failUsage("usage: sq dep add <issue-id> <depends-on-id> [--json]")
		}
		if err := store.AddDependency(db, args[1], args[2], "blocks"); err != nil {
			return failRuntime(err.Error())
		}
		return printJSON(map[string]any{"issue_id": args[1], "depends_on_id": args[2], "type": "blocks"})
	case "remove", "rm":
		if len(args) < 3 {
			return failUsage("usage: sq dep remove <issue-id> <depends-on-id> [--json]")
		}
		if err := store.RemoveDependency(db, args[1], args[2]); err != nil {
			return failRuntime(err.Error())
		}
		return printJSON(map[string]any{"issue_id": args[1], "depends_on_id": args[2], "removed": true})
	case "list":
		if len(args) < 2 {
			return failUsage("usage: sq dep list <issue-id> [--json]")
		}
		deps, err := store.ListDependencies(db, args[1])
		if err != nil {
			return failRuntime(err.Error())
		}
		return printJSON(deps)
	default:
		return failUsage("unknown dep subcommand: " + sub)
	}
}

func cmdComments(args []string) int {
	if len(args) == 0 {
		return failUsage("usage: sq comments <issue-id> [--json] OR sq comments add <issue-id> <text> [--json]")
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()

	if args[0] == "add" {
		if len(args) < 3 {
			return failUsage("usage: sq comments add <issue-id> <text> [--json]")
		}
		issueID := args[1]
		body := args[2]
		author := strings.TrimSpace(os.Getenv("BD_ACTOR"))
		c, err := store.AddComment(db, issueID, author, body)
		if err != nil {
			return failRuntime(err.Error())
		}
		return printJSON(c)
	}

	issueID := args[0]
	comments, err := store.ListComments(db, issueID)
	if err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(comments)
}

func cmdTodo(args []string) int {
	sub := "list"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = args[0]
		args = args[1:]
	}

	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()

	switch sub {
	case "list":
		all, err := store.ListTasks(db)
		if err != nil {
			return failRuntime(err.Error())
		}
		tasks := make([]store.Task, 0)
		for _, t := range all {
			if t.IssueType == "task" && t.Status == "open" {
				tasks = append(tasks, t)
			}
		}
		if len(tasks) == 0 {
			return printJSON(nil)
		}
		return printJSON(tasks)
	case "add":
		if len(args) == 0 {
			return failUsage("usage: sq todo add <title> [--priority N] [--description TEXT] [--json]")
		}
		in := store.CreateInput{Title: args[0], IssueType: "task", Priority: 2}
		creator := strings.TrimSpace(os.Getenv("BD_ACTOR"))
		if creator == "" {
			creator = strings.TrimSpace(os.Getenv("USER"))
		}
		in.Creator = creator
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--priority":
				if i+1 < len(args) {
					p, err := strconv.Atoi(args[i+1])
					if err != nil {
						return failUsage("invalid --priority")
					}
					in.Priority = p
					i++
				}
			case "--description":
				if i+1 < len(args) {
					in.Description = args[i+1]
					i++
				}
			case "--json":
				// accepted, no-op
			default:
				if strings.HasPrefix(args[i], "-") {
					return failUsage("unknown flag: " + args[i])
				}
			}
		}
		t, err := store.CreateTask(db, in)
		if err != nil {
			return failRuntime(err.Error())
		}
		return printJSON(t)
	case "done":
		if len(args) == 0 {
			return failUsage("usage: sq todo done <id> [--reason TEXT] [--json]")
		}
		id := args[0]
		reason := "Completed"
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--reason":
				if i+1 < len(args) {
					reason = args[i+1]
					i++
				}
			case "--json":
				// accepted, no-op
			default:
				if strings.HasPrefix(args[i], "-") {
					return failUsage("unknown flag: " + args[i])
				}
			}
		}
		t, err := store.CloseTask(db, id, reason)
		if err != nil {
			return failRuntime(err.Error())
		}
		return printJSON(t)
	default:
		return failUsage("unknown todo subcommand: " + sub)
	}
}

func cmdChildren(args []string) int {
	if len(args) == 0 {
		return failUsage("usage: sq children <parent-id> [--json]")
	}
	parentID := args[0]
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	items, err := store.ListChildren(db, parentID)
	if err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(items)
}

func cmdBlocked(args []string) int {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			continue
		case "--parent":
			if i+1 < len(args) {
				i++
			}
		default:
			if strings.HasPrefix(args[i], "-") {
				return failUsage("unknown flag: " + args[i])
			}
		}
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	items, err := store.ListBlocked(db)
	if err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(items)
}

func cmdDuplicate(args []string) int {
	if len(args) == 0 {
		return failUsage("usage: sq duplicate <id> --of <canonical-id> [--json]")
	}
	id := args[0]
	canonical := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--of":
			if i+1 < len(args) {
				canonical = args[i+1]
				i++
			}
		case "--json":
			// accepted
		default:
			if strings.HasPrefix(args[i], "-") {
				return failUsage("unknown flag: " + args[i])
			}
		}
	}
	if canonical == "" {
		return failUsage("--of is required")
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	if err := store.AddDependency(db, id, canonical, "duplicates"); err != nil {
		return failRuntime(err.Error())
	}
	if _, err := store.CloseTask(db, id, "Duplicate of "+canonical); err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(map[string]any{"canonical": canonical, "duplicate": id, "status": "closed"})
}

func cmdSupersede(args []string) int {
	if len(args) == 0 {
		return failUsage("usage: sq supersede <id> --with <replacement-id> [--json]")
	}
	id := args[0]
	replacement := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--with":
			if i+1 < len(args) {
				replacement = args[i+1]
				i++
			}
		case "--json":
			// accepted
		default:
			if strings.HasPrefix(args[i], "-") {
				return failUsage("unknown flag: " + args[i])
			}
		}
	}
	if replacement == "" {
		return failUsage("--with is required")
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	if err := store.AddDependency(db, id, replacement, "supersedes"); err != nil {
		return failRuntime(err.Error())
	}
	if _, err := store.CloseTask(db, id, "Superseded by "+replacement); err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(map[string]any{"replacement": replacement, "superseded": id, "status": "closed"})
}

func cmdTypes(args []string) int {
	for _, a := range args {
		if a == "--json" {
			continue
		}
		if strings.HasPrefix(a, "-") {
			return failUsage("unknown flag: " + a)
		}
	}
	return printJSON(map[string]any{"core_types": []map[string]string{
		{"name": "task", "description": "General work item (default)"},
		{"name": "bug", "description": "Bug report or defect"},
		{"name": "feature", "description": "New feature or enhancement"},
		{"name": "chore", "description": "Maintenance or housekeeping"},
		{"name": "epic", "description": "Large body of work spanning multiple issues"},
		{"name": "decision", "description": "Architecture decision record (ADR)"},
	}})
}

func cmdQuery(args []string) int {
	if len(args) == 0 {
		return failUsage("query expression required")
	}
	filtered := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--json" || a == "-a" || a == "--all" {
			continue
		}
		if strings.HasPrefix(a, "--sort") || a == "--reverse" || a == "--long" || a == "--parse-only" || a == "--limit" {
			continue
		}
		filtered = append(filtered, a)
	}
	expr := strings.TrimSpace(strings.Join(filtered, " "))
	if expr == "" {
		return failUsage("query expression required")
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	items, err := store.QueryTasks(db, expr)
	if err != nil {
		return failUsage(err.Error())
	}
	return printJSON(items)
}

func cmdSearch(args []string) int {
	query := ""
	limit := 50
	if len(args) > 0 {
		query = args[0]
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--query":
			if i+1 < len(args) {
				query = args[i+1]
				i++
			}
		case "--limit", "-n":
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil {
					limit = n
				}
				i++
			}
		case "--json", "--status", "--sort", "--reverse", "--long":
			// accepted compatibility flags
		default:
			if strings.HasPrefix(args[i], "-") {
				// ignore unsupported compatibility flags for now
				continue
			}
		}
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	items, err := store.SearchTasks(db, query, limit)
	if err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(items)
}

func cmdCount(args []string) int {
	status := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--status" || args[i] == "-s" {
			if i+1 < len(args) {
				status = args[i+1]
				i++
			}
		}
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	n, err := store.CountTasks(db, status)
	if err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(map[string]any{"count": n})
}

func cmdStatus() int {
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	s, err := store.StatusSummary(db)
	if err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(s)
}

// Run executes the sq CLI with pre-sliced args (excluding argv[0])
// and returns a process exit code.
func Run(args []string) int {
	if len(args) < 1 {
		usage()
		return 2
	}

	switch args[0] {
	case "-h", "--help", "help":
		usage()
		return 0
	case "init":
		return cmdInit()
	case "ready":
		return cmdReady()
	case "create":
		return cmdCreate(args[1:])
	case "show":
		return cmdShow(args[1:])
	case "list":
		return cmdList(args[1:])
	case "update":
		return cmdUpdate(args[1:])
	case "close":
		return cmdClose(args[1:])
	case "reopen":
		return cmdReopen(args[1:])
	case "delete":
		return cmdDelete(args[1:])
	case "label":
		return cmdLabel(args[1:])
	case "dep":
		return cmdDep(args[1:])
	case "comments":
		return cmdComments(args[1:])
	case "todo":
		return cmdTodo(args[1:])
	case "children":
		return cmdChildren(args[1:])
	case "blocked":
		return cmdBlocked(args[1:])
	case "duplicate":
		return cmdDuplicate(args[1:])
	case "supersede":
		return cmdSupersede(args[1:])
	case "types":
		return cmdTypes(args[1:])
	case "query":
		return cmdQuery(args[1:])
	case "search":
		return cmdSearch(args[1:])
	case "count":
		return cmdCount(args[1:])
	case "status", "stats":
		return cmdStatus()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		usage()
		return 2
	}
}
