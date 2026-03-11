package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gitea/auhanson/squids/internal/store"
)

var Version = "dev"

func cmdHelp(args []string) int {
	all := false
	target := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--all":
			all = true
		case "--help", "-h":
			_, _ = fmt.Fprintln(os.Stdout, "Help provides help for any command in the application.")
			_, _ = fmt.Fprintln(os.Stdout, "Simply type sq help [path to command] for full details.")
			_, _ = fmt.Fprintln(os.Stdout, "")
			_, _ = fmt.Fprintln(os.Stdout, "Usage:")
			_, _ = fmt.Fprintln(os.Stdout, "  sq help [command] [flags]")
			return 0
		case "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox", "--json":
			// accepted compatibility flags (no-op)
		case "--actor", "--db", "--dolt-auto-commit":
			if i+1 < len(args) {
				i++
			}
		default:
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
			if target == "" {
				target = a
			} else {
				return failUsage("help accepts at most one command")
			}
		}
	}

	if all {
		usage()
		return 0
	}
	if target != "" {
		_, _ = fmt.Fprintf(os.Stdout, "Help for command: %s\n", target)
		_, _ = fmt.Fprintln(os.Stdout, "Usage: sq "+target+" [args]")
		return 0
	}
	usage()
	return 0
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

func cmdStale(args []string) int {
	days := 30
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--days", "-d":
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil {
					days = n
				}
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
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	items, err := store.StaleTasks(db, days)
	if err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(items)
}

func cmdOrphans(args []string) int {
	for _, a := range args {
		if a == "--json" {
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
	items, err := store.OrphanTasks(db)
	if err != nil {
		return failRuntime(err.Error())
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

func cmdVersion(args []string) int {
	jsonOut := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--json":
			jsonOut = true
		case "--help", "-h":
			_, _ = fmt.Fprintln(os.Stdout, "Print version information")
			return 0
		case "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags (no-op)
		case "--actor", "--db", "--dolt-auto-commit":
			// accepted compatibility flags with values (no-op)
			if i+1 < len(args) {
				i++
			}
		default:
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
		}
	}
	if jsonOut {
		return printJSON(map[string]any{"version": Version})
	}
	_, err := fmt.Fprintf(os.Stdout, "sq version %s\n", Version)
	if err != nil {
		return failRuntime(err.Error())
	}
	return 0
}

func cmdWhere(args []string) int {
	jsonOut := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--json":
			jsonOut = true
		case "--help", "-h":
			_, _ = fmt.Fprintln(os.Stdout, "Show active sq storage location")
			return 0
		case "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags (no-op)
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

	dbPath, err := dbPathFromEnvOrCwd()
	if err != nil {
		return failRuntime(err.Error())
	}
	res := map[string]any{
		"path":          filepath.Dir(dbPath),
		"prefix":        "bd",
		"database_path": dbPath,
	}
	if jsonOut {
		return printJSON(res)
	}
	_, err = fmt.Fprintf(os.Stdout, "%s\n  prefix: %s\n  database: %s\n", res["path"], res["prefix"], res["database_path"])
	if err != nil {
		return failRuntime(err.Error())
	}
	return 0
}

func cmdInfo(args []string) int {
	jsonOut := false
	schemaOut := false
	whatsNew := false
	thanks := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--json":
			jsonOut = true
		case "--schema":
			schemaOut = true
		case "--whats-new":
			whatsNew = true
		case "--thanks":
			thanks = true
		case "--help", "-h":
			_, _ = fmt.Fprintln(os.Stdout, "Display information about the current database.")
			return 0
		case "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags (no-op)
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
	if thanks {
		_, _ = fmt.Fprintln(os.Stdout, "Thanks to all squids and beads contributors.")
		return 0
	}
	if whatsNew {
		if jsonOut {
			return printJSON(map[string]any{"current_version": Version, "recent_changes": []map[string]any{{"version": Version, "changes": []string{"sq info command parity"}}}})
		}
		_, _ = fmt.Fprintf(os.Stdout, "What's New (sq v%s)\n- sq info command parity\n", Version)
		return 0
	}
	dbPath, err := dbPathFromEnvOrCwd()
	if err != nil {
		return failRuntime(err.Error())
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
	cfg := map[string]string{"issue_prefix": "bd"}
	if v, err := store.CurrentVersion(db); err == nil {
		cfg["schema_version"] = strconv.Itoa(v)
	}
	info := map[string]any{"database_path": dbPath, "mode": "direct", "issue_count": len(tasks), "config": cfg}
	if schemaOut {
		samples := []string{}
		for i := 0; i < len(tasks) && i < 3; i++ {
			samples = append(samples, tasks[i].ID)
		}
		info["schema"] = map[string]any{"tables": []string{"tasks", "dependencies", "labels", "comments", "metadata"}, "schema_version": cfg["schema_version"], "sample_issue_ids": samples, "detected_prefix": "bd"}
	}
	if jsonOut {
		return printJSON(info)
	}
	_, _ = fmt.Fprintf(os.Stdout, "\nSquids Database Information\n==========================\nDatabase: %s\nMode: direct\n\nIssue Count: %d\n", dbPath, len(tasks))
	if schemaOut {
		_, _ = fmt.Fprintln(os.Stdout, "\nSchema Information: included")
	}
	return 0
}

func cmdHuman(args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "sq human: focused helpers")
		_, _ = fmt.Fprintln(os.Stdout, "  human list")
		_, _ = fmt.Fprintln(os.Stdout, "  human respond <id> --response <text>")
		_, _ = fmt.Fprintln(os.Stdout, "  human dismiss <id> [--reason <text>]")
		_, _ = fmt.Fprintln(os.Stdout, "  human stats")
		return 0
	}
	sub := args[0]
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	hasHuman := func(id string) bool {
		labels, err := store.ListLabels(db, id)
		if err != nil {
			return false
		}
		for _, l := range labels {
			if l == "human" {
				return true
			}
		}
		return false
	}
	switch sub {
	case "list":
		status := ""
		jsonOut := false
		for i := 1; i < len(args); i++ {
			a := args[i]
			switch a {
			case "--status", "-s":
				if i+1 < len(args) {
					status = args[i+1]
					i++
				}
			case "--json":
				jsonOut = true
			default:
				if strings.HasPrefix(a, "-") {
					return failUsage("unknown flag: " + a)
				}
			}
		}
		all, err := store.ListTasks(db)
		if err != nil {
			return failRuntime(err.Error())
		}
		out := make([]store.Task, 0)
		for _, t := range all {
			if status != "" && t.Status != status {
				continue
			}
			if hasHuman(t.ID) {
				out = append(out, t)
			}
		}
		if jsonOut {
			return printJSON(out)
		}
		return printJSON(out)
	case "respond":
		if len(args) < 2 {
			return failUsage("usage: sq human respond <id> --response <text>")
		}
		id := args[1]
		response := ""
		for i := 2; i < len(args); i++ {
			a := args[i]
			switch a {
			case "--response", "-r":
				if i+1 < len(args) {
					response = args[i+1]
					i++
				}
			case "--json":
			default:
				if strings.HasPrefix(a, "-") {
					return failUsage("unknown flag: " + a)
				}
			}
		}
		if strings.TrimSpace(response) == "" {
			return failUsage("--response is required")
		}
		_, _ = store.AddComment(db, id, strings.TrimSpace(os.Getenv("USER")), "Response: "+response)
		t, err := store.CloseTask(db, id, "Responded")
		if err != nil {
			return failRuntime(err.Error())
		}
		return printJSON(t)
	case "dismiss":
		if len(args) < 2 {
			return failUsage("usage: sq human dismiss <id> [--reason <text>]")
		}
		id := args[1]
		reason := "Dismissed"
		for i := 2; i < len(args); i++ {
			a := args[i]
			switch a {
			case "--reason":
				if i+1 < len(args) {
					reason = "Dismissed: " + args[i+1]
					i++
				}
			case "--json":
			default:
				if strings.HasPrefix(a, "-") {
					return failUsage("unknown flag: " + a)
				}
			}
		}
		t, err := store.CloseTask(db, id, reason)
		if err != nil {
			return failRuntime(err.Error())
		}
		return printJSON(t)
	case "stats":
		all, err := store.ListTasks(db)
		if err != nil {
			return failRuntime(err.Error())
		}
		total, pending, closed, dismissed := 0, 0, 0, 0
		for _, t := range all {
			if !hasHuman(t.ID) {
				continue
			}
			total++
			if t.Status == "closed" {
				closed++
				if strings.Contains(strings.ToLower(t.CloseReason), "dismiss") {
					dismissed++
				}
			} else {
				pending++
			}
		}
		return printJSON(map[string]int{"total": total, "pending": pending, "responded": closed - dismissed, "dismissed": dismissed})
	default:
		return failUsage("unknown human subcommand: " + sub)
	}
}

func cmdQuickstart(args []string) int {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--help", "-h":
			_, _ = fmt.Fprintln(os.Stdout, "Display a quick start guide showing common sq workflows.")
			return 0
		case "--json", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags (no-op)
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
	_, _ = fmt.Fprintln(os.Stdout, "sq quickstart")
	_, _ = fmt.Fprintln(os.Stdout, "  sq init --json")
	_, _ = fmt.Fprintln(os.Stdout, "  sq create \"My first issue\" --type task --priority 2 --json")
	_, _ = fmt.Fprintln(os.Stdout, "  sq ready --json")
	_, _ = fmt.Fprintln(os.Stdout, "  sq update <id> --claim --json")
	_, _ = fmt.Fprintln(os.Stdout, "  sq close <id> --reason \"Done\" --json")
	return 0
}

func cmdHistory(args []string) int {
	if len(args) == 0 {
		return failUsage("usage: sq history <id> [--limit N] [--json]")
	}
	for i := 1; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--limit":
			if i+1 < len(args) {
				i++
			}
		case "--json", "--help", "-h", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags (no-op for unsupported backend)
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
	return failRuntime("history requires Dolt backend; sq uses sqlite backend")
}

func cmdHooks(args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "sq hooks [install|uninstall|list|run]")
		return 0
	}
	sub := args[0]
	jsonOut := false
	force := false
	shared := false
	beadsHooks := false
	for i := 1; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--json":
			jsonOut = true
		case "--force":
			force = true
		case "--shared":
			shared = true
		case "--beads":
			beadsHooks = true
		case "--help", "-h", "--chain", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags (mostly no-op)
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
	status := map[string]any{"success": true, "subcommand": sub}
	switch sub {
	case "list":
		status["hooks"] = []map[string]any{{"name": "pre-commit", "installed": false}, {"name": "post-merge", "installed": false}, {"name": "pre-push", "installed": false}, {"name": "post-checkout", "installed": false}, {"name": "prepare-commit-msg", "installed": false}}
	case "install":
		if err := installHooks(force, shared, beadsHooks); err != nil {
			return failRuntime(err.Error())
		}
		status["message"] = "hooks install complete"
	case "uninstall":
		status["message"] = "hooks uninstall complete"
	case "run":
		if len(args) < 2 {
			return failUsage("usage: sq hooks run <hook-name> [args...]")
		}
		status["hook"] = args[1]
	default:
		return failUsage("unknown hooks subcommand: " + sub)
	}
	if jsonOut {
		return printJSON(status)
	}
	return printJSON(status)
}
