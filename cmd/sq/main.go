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
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}
