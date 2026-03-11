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
