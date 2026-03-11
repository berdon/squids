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

type result struct {
	Command       string `json:"command"`
	OK            bool   `json:"ok"`
	Message       string `json:"message,omitempty"`
	DBPath        string `json:"db_path,omitempty"`
	SchemaVersion int    `json:"schema_version,omitempty"`
}

func printJSON(v any) int {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode json: %v\n", err)
		return 1
	}
	return 0
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
	if err := store.Init(db); err != nil {
		_ = db.Close()
		return nil, dbPath, err
	}
	return db, dbPath, nil
}

func cmdInit() int {
	dbPath, err := dbPathFromEnvOrCwd()
	if err != nil {
		return printJSON(result{Command: "init", OK: false, Message: err.Error()})
	}
	db, err := store.Open(dbPath)
	if err != nil {
		return printJSON(result{Command: "init", OK: false, Message: err.Error(), DBPath: dbPath})
	}
	defer db.Close()
	if err := store.Init(db); err != nil {
		return printJSON(result{Command: "init", OK: false, Message: err.Error(), DBPath: dbPath})
	}
	v, err := store.CurrentVersion(db)
	if err != nil {
		return printJSON(result{Command: "init", OK: false, Message: err.Error(), DBPath: dbPath})
	}
	return printJSON(result{Command: "init", OK: true, DBPath: dbPath, SchemaVersion: v})
}

func cmdReady() int {
	dbPath, err := dbPathFromEnvOrCwd()
	if err != nil {
		return printJSON([]result{{Command: "ready", OK: false, Message: err.Error()}})
	}
	db, err := store.Open(dbPath)
	if err != nil {
		return printJSON([]result{{Command: "ready", OK: false, Message: err.Error(), DBPath: dbPath}})
	}
	defer db.Close()
	if err := store.Init(db); err != nil {
		return printJSON([]result{{Command: "ready", OK: false, Message: err.Error(), DBPath: dbPath}})
	}
	if err := store.Ping(db); err != nil {
		return printJSON([]result{{Command: "ready", OK: false, Message: err.Error(), DBPath: dbPath}})
	}
	v, err := store.CurrentVersion(db)
	if err != nil {
		return printJSON([]result{{Command: "ready", OK: false, Message: err.Error(), DBPath: dbPath}})
	}
	return printJSON([]result{{Command: "ready", OK: true, DBPath: dbPath, SchemaVersion: v}})
}

func cmdCreate(args []string) int {
	if len(args) == 0 {
		return printJSON(result{Command: "create", OK: false, Message: "title is required"})
	}
	in := store.CreateInput{Title: args[0], IssueType: "task"}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 < len(args) { in.IssueType = args[i+1]; i++ }
		case "--priority":
			if i+1 < len(args) { if p, err := strconv.Atoi(args[i+1]); err == nil { in.Priority = p }; i++ }
		case "--description":
			if i+1 < len(args) { in.Description = args[i+1]; i++ }
		}
	}
	db, dbPath, err := openTaskDB()
	if err != nil { return printJSON(result{Command:"create",OK:false,Message:err.Error(),DBPath:dbPath}) }
	defer db.Close()
	t, err := store.CreateTask(db, in)
	if err != nil { return printJSON(result{Command:"create",OK:false,Message:err.Error(),DBPath:dbPath}) }
	return printJSON(t)
}

func cmdShow(args []string) int {
	if len(args) == 0 { return printJSON(result{Command:"show",OK:false,Message:"id is required"}) }
	db, _, err := openTaskDB()
	if err != nil { return printJSON(result{Command:"show",OK:false,Message:err.Error()}) }
	defer db.Close()
	t, err := store.ShowTask(db, args[0])
	if err != nil { fmt.Fprintln(os.Stderr, err.Error()); return 1 }
	return printJSON(t)
}

func cmdList() int {
	db, dbPath, err := openTaskDB()
	if err != nil { return printJSON(result{Command:"list",OK:false,Message:err.Error(),DBPath:dbPath}) }
	defer db.Close()
	tasks, err := store.ListTasks(db)
	if err != nil { return printJSON(result{Command:"list",OK:false,Message:err.Error(),DBPath:dbPath}) }
	return printJSON(tasks)
}

func cmdUpdate(args []string) int {
	if len(args) == 0 { return printJSON(result{Command:"update",OK:false,Message:"id is required"}) }
	id := args[0]
	in := store.UpdateInput{SetMetadata: map[string]string{}}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--status":
			if i+1 < len(args) { v := args[i+1]; in.Status = &v; i++ }
		case "--assignee":
			if i+1 < len(args) { v := args[i+1]; in.Assignee = &v; i++ }
		case "--add-label":
			if i+1 < len(args) { in.AddLabels = append(in.AddLabels, args[i+1]); i++ }
		case "--set-metadata":
			if i+1 < len(args) { kv := strings.SplitN(args[i+1], "=", 2); if len(kv) == 2 { in.SetMetadata[kv[0]] = kv[1] }; i++ }
		case "--claim":
			in.Claim = true
		}
	}
	db, dbPath, err := openTaskDB()
	if err != nil { return printJSON(result{Command:"update",OK:false,Message:err.Error(),DBPath:dbPath}) }
	defer db.Close()
	t, err := store.UpdateTask(db, id, in)
	if err != nil { return printJSON(result{Command:"update",OK:false,Message:err.Error(),DBPath:dbPath}) }
	return printJSON(t)
}

func cmdClose(args []string) int {
	if len(args) == 0 { return printJSON(result{Command:"close",OK:false,Message:"id is required"}) }
	id := args[0]
	reason := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--reason" && i+1 < len(args) { reason = args[i+1]; i++ }
	}
	db, dbPath, err := openTaskDB()
	if err != nil { return printJSON(result{Command:"close",OK:false,Message:err.Error(),DBPath:dbPath}) }
	defer db.Close()
	t, err := store.CloseTask(db, id, reason)
	if err != nil { return printJSON(result{Command:"close",OK:false,Message:err.Error(),DBPath:dbPath}) }
	return printJSON(t)
}

func main() {
	if len(os.Args) < 2 {
		usage(); os.Exit(2)
	}
	switch os.Args[1] {
	case "-h", "--help", "help":
		usage(); os.Exit(0)
	case "init":
		os.Exit(cmdInit())
	case "ready":
		os.Exit(cmdReady())
	case "create":
		os.Exit(cmdCreate(os.Args[2:]))
	case "show":
		os.Exit(cmdShow(os.Args[2:]))
	case "list":
		os.Exit(cmdList())
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
