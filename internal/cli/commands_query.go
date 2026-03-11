package cli

import (
	"strconv"
	"strings"

	"gitea/auhanson/squids/internal/store"
)

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
