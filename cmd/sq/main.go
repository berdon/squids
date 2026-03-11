package main

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
	fmt.Println("  query   Query tasks")
	fmt.Println("  search  Search tasks")
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
		if len(args) < 3 { return failUsage("usage: sq label add <id> <label> [--json]") }
		t, err := store.AddLabel(db, args[1], args[2])
		if err != nil { return failRuntime(err.Error()) }
		return printJSON(t)
	case "remove":
		if len(args) < 3 { return failUsage("usage: sq label remove <id> <label> [--json]") }
		t, err := store.RemoveLabel(db, args[1], args[2])
		if err != nil { return failRuntime(err.Error()) }
		return printJSON(t)
	case "list":
		if len(args) < 2 { return failUsage("usage: sq label list <id> [--json]") }
		labels, err := store.ListLabels(db, args[1])
		if err != nil { return failRuntime(err.Error()) }
		return printJSON(labels)
	case "list-all":
		labels, err := store.ListAllLabels(db)
		if err != nil { return failRuntime(err.Error()) }
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
		if len(args) < 3 { return failUsage("usage: sq dep add <issue-id> <depends-on-id> [--json]") }
		if err := store.AddDependency(db, args[1], args[2], "blocks"); err != nil { return failRuntime(err.Error()) }
		return printJSON(map[string]any{"issue_id": args[1], "depends_on_id": args[2], "type": "blocks"})
	case "remove", "rm":
		if len(args) < 3 { return failUsage("usage: sq dep remove <issue-id> <depends-on-id> [--json]") }
		if err := store.RemoveDependency(db, args[1], args[2]); err != nil { return failRuntime(err.Error()) }
		return printJSON(map[string]any{"issue_id": args[1], "depends_on_id": args[2], "removed": true})
	case "list":
		if len(args) < 2 { return failUsage("usage: sq dep list <issue-id> [--json]") }
		deps, err := store.ListDependencies(db, args[1])
		if err != nil { return failRuntime(err.Error()) }
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

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "-h", "--help", "help":
		usage()
		os.Exit(0)
	case "init":
		os.Exit(cmdInit())
	case "ready":
		os.Exit(cmdReady())
	case "create":
		os.Exit(cmdCreate(os.Args[2:]))
	case "show":
		os.Exit(cmdShow(os.Args[2:]))
	case "list":
		os.Exit(cmdList(os.Args[2:]))
	case "update":
		os.Exit(cmdUpdate(os.Args[2:]))
	case "close":
		os.Exit(cmdClose(os.Args[2:]))
	case "reopen":
		os.Exit(cmdReopen(os.Args[2:]))
	case "delete":
		os.Exit(cmdDelete(os.Args[2:]))
	case "label":
		os.Exit(cmdLabel(os.Args[2:]))
	case "dep":
		os.Exit(cmdDep(os.Args[2:]))
	case "comments":
		os.Exit(cmdComments(os.Args[2:]))
	case "query":
		os.Exit(cmdQuery(os.Args[2:]))
	case "search":
		os.Exit(cmdSearch(os.Args[2:]))
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}
