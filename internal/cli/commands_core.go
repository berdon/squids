package cli

import (
	"os"
	"strconv"
	"strings"

	"gitea/auhanson/squids/internal/store"
)

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
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	items, err := store.ReadyTasks(db)
	if err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(items)
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
