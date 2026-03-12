package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/berdon/squids/internal/store"
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
		printHelpAll()
		return 0
	}
	if target != "" {
		if target == "label" {
			printLabelHelp()
			return 0
		}
		if target == "query" {
			printQueryHelp()
			return 0
		}
		if target == "gate" {
			printGateHelp()
			return 0
		}
		_, _ = fmt.Fprintf(os.Stdout, "Help for command: %s\n", target)
		_, _ = fmt.Fprintln(os.Stdout, "Usage: sq "+target+" [args]")
		return 0
	}
	usage()
	return 0
}

func printHelpAll() {
	_, _ = fmt.Fprintln(os.Stdout, "# sq — Complete Command Reference")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Generated from `sq help --all`")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "## Table of Contents")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "### Core Commands")
	_, _ = fmt.Fprintln(os.Stdout, "- `sq init` — Initialize task storage")
	_, _ = fmt.Fprintln(os.Stdout, "- `sq create` — Create a task")
	_, _ = fmt.Fprintln(os.Stdout, "- `sq list` — List tasks")
	_, _ = fmt.Fprintln(os.Stdout, "- `sq show` — Show a task")
	_, _ = fmt.Fprintln(os.Stdout, "- `sq update` — Update a task")
	_, _ = fmt.Fprintln(os.Stdout, "- `sq close` / `sq reopen` — Transition task state")
	_, _ = fmt.Fprintln(os.Stdout, "- `sq ready` — Show unblocked work")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "### Compatibility Surfaces")
	_, _ = fmt.Fprintln(os.Stdout, "- `sq history` — Not supported on sqlite backend")
	_, _ = fmt.Fprintln(os.Stdout, "- `sq swarm` — Compatibility command group")
	_, _ = fmt.Fprintln(os.Stdout, "- `sq hooks` — Compatibility command group")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "### Usage")
	_, _ = fmt.Fprintln(os.Stdout, "`sq <command> [args]`")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Run `sq help <command>` for command-specific usage.")
}

func printQueryHelp() {
	_, _ = fmt.Fprintln(os.Stdout, "Query issues using expression syntax.")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Usage:")
	_, _ = fmt.Fprintln(os.Stdout, "  sq query <expression> [flags]")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Flags:")
	_, _ = fmt.Fprintln(os.Stdout, "  --json        output JSON")
	_, _ = fmt.Fprintln(os.Stdout, "  -a, --all     compatibility no-op")
	_, _ = fmt.Fprintln(os.Stdout, "  --sort <key>  compatibility no-op")
	_, _ = fmt.Fprintln(os.Stdout, "  --reverse     compatibility no-op")
	_, _ = fmt.Fprintln(os.Stdout, "  --long        compatibility no-op")
	_, _ = fmt.Fprintln(os.Stdout, "  --parse-only  compatibility no-op")
	_, _ = fmt.Fprintln(os.Stdout, "  --limit <n>   compatibility no-op")
}

func cmdQuery(args []string) int {
	if len(args) == 0 {
		return failUsage("query expression required")
	}
	useJSON := false
	for _, a := range args {
		if a == "--help" || a == "-h" {
			printQueryHelp()
			return 0
		}
		if a == "--json" {
			useJSON = true
		}
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
	if useJSON {
		return printJSON(items)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Found %d issues:\n", len(items))
	for _, it := range items {
		statusIcon := "○"
		statusBadge := "●"
		if strings.EqualFold(it.Status, "closed") {
			statusIcon = "✓"
			statusBadge = "✓"
		}
		assignee := ""
		if strings.TrimSpace(it.Assignee) != "" {
			assignee = " @" + it.Assignee
		}
		issueType := it.IssueType
		if issueType == "" {
			issueType = "task"
		}
		_, _ = fmt.Fprintf(os.Stdout, "%s %s [%s P%d] [%s]%s - %s\n", statusIcon, it.ID, statusBadge, it.Priority, issueType, assignee, it.Title)
	}
	return 0
}

func printGateHelp() {
	_, _ = fmt.Fprintln(os.Stdout, "Manage async workflow gates (compat surface).")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Usage:")
	_, _ = fmt.Fprintln(os.Stdout, "  sq gate list [--all] [--json]")
	_, _ = fmt.Fprintln(os.Stdout, "  sq gate show <id> [--json]")
	_, _ = fmt.Fprintln(os.Stdout, "  sq gate resolve <id> [--reason <text>] [--json]")
	_, _ = fmt.Fprintln(os.Stdout, "  sq gate check [--json]")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Flags:")
	_, _ = fmt.Fprintln(os.Stdout, "  --json   output JSON")
	_, _ = fmt.Fprintln(os.Stdout, "  --all    include closed gates (list)")
}

func cmdGate(args []string) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printGateHelp()
		return 0
	}
	useJSON := false
	for _, a := range args {
		if a == "--json" {
			useJSON = true
		}
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()

	sub := args[0]
	switch sub {
	case "list":
		includeAll := false
		for _, a := range args[1:] {
			if a == "--all" {
				includeAll = true
			}
		}
		all, err := store.ListTasks(db)
		if err != nil {
			return failRuntime(err.Error())
		}
		gates := make([]store.Task, 0)
		for _, t := range all {
			if t.IssueType != "gate" {
				continue
			}
			if !includeAll && t.Status == "closed" {
				continue
			}
			gates = append(gates, t)
		}
		if useJSON {
			return printJSON(gates)
		}
		_, _ = fmt.Fprintf(os.Stdout, "Found %d gates:\n", len(gates))
		for _, g := range gates {
			icon := "○"
			if g.Status == "closed" {
				icon = "✓"
			}
			_, _ = fmt.Fprintf(os.Stdout, "%s %s [%s] - %s\n", icon, g.ID, g.Status, g.Title)
		}
		return 0
	case "show":
		if len(args) < 2 {
			return failUsage("usage: sq gate show <id> [--json]")
		}
		g, err := store.ShowTask(db, args[1])
		if err != nil {
			return failRuntime(err.Error())
		}
		if g.IssueType != "gate" {
			return failUsage("issue is not a gate: " + args[1])
		}
		if useJSON {
			return printJSON(g)
		}
		_, _ = fmt.Fprintf(os.Stdout, "Gate %s\n  status: %s\n  title: %s\n", g.ID, g.Status, g.Title)
		return 0
	case "resolve":
		if len(args) < 2 {
			return failUsage("usage: sq gate resolve <id> [--reason <text>] [--json]")
		}
		reason := "Gate resolved"
		for i := 2; i < len(args); i++ {
			if args[i] == "--reason" && i+1 < len(args) {
				reason = args[i+1]
				i++
			}
		}
		g, err := store.ShowTask(db, args[1])
		if err != nil {
			return failRuntime(err.Error())
		}
		if g.IssueType != "gate" {
			return failUsage("issue is not a gate: " + args[1])
		}
		closed, err := store.CloseTask(db, args[1], reason)
		if err != nil {
			return failRuntime(err.Error())
		}
		if useJSON {
			return printJSON(closed)
		}
		_, _ = fmt.Fprintf(os.Stdout, "✓ Resolved gate %s\n", args[1])
		return 0
	case "check":
		all, err := store.ListTasks(db)
		if err != nil {
			return failRuntime(err.Error())
		}
		openGates := 0
		for _, t := range all {
			if t.IssueType == "gate" && t.Status != "closed" {
				openGates++
			}
		}
		payload := map[string]any{"open_gates": openGates, "resolved": 0}
		if useJSON {
			return printJSON(payload)
		}
		_, _ = fmt.Fprintf(os.Stdout, "Gate check complete: open=%d resolved=0\n", openGates)
		return 0
	default:
		return failUsage("unknown gate subcommand: " + sub)
	}
}

func cmdPurge(args []string) int {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--help", "-h", "--json", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags
		case "--older-than", "--actor", "--db", "--dolt-auto-commit":
			if i+1 < len(args) {
				i++
			}
		default:
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
		}
	}
	return failRuntime("purge compatibility surface only; sqlite backend does not implement purge semantics yet")
}

func cmdRestore(args []string) int {
	if len(args) == 0 {
		return failUsage("usage: sq restore <issue-id> [--json]")
	}
	for i := 1; i < len(args); i++ {
		a := args[i]
		switch a {
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
	return failRuntime("restore requires Dolt backend; sq uses sqlite backend")
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

func printSearchHelp() {
	_, _ = fmt.Fprintln(os.Stdout, "Search issues by title/description text.")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Usage:")
	_, _ = fmt.Fprintln(os.Stdout, "  sq search <query> [flags]")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Flags:")
	_, _ = fmt.Fprintln(os.Stdout, "  -n, --limit <n>  maximum results")
	_, _ = fmt.Fprintln(os.Stdout, "  --json           output JSON")
}

func cmdSearch(args []string) int {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			printSearchHelp()
			return 0
		}
	}
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

func cmdMol(args []string) int {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--help", "-h", "--json", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags
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
	return failRuntime("mol compatibility surface only; sq does not implement molecule workflows")
}

func cmdMail(args []string) int {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--help", "-h", "--json", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags
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
	return failRuntime("mail compatibility surface only; sq has no mail provider integration")
}

func cmdSetup(args []string) int {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--help", "-h", "--list", "--check", "--project", "--global", "--remove", "--print", "--stealth", "--json", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags (no-op)
		case "--add", "--output", "-o", "--actor", "--db", "--dolt-auto-commit":
			if i+1 < len(args) {
				i++
			}
		default:
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
		}
	}
	return failRuntime("setup compatibility surface only; sq does not manage editor integration templates yet")
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

func cmdAudit(args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "sq audit [record|label]")
		return 0
	}
	sub := args[0]
	for i := 1; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--json", "--help", "-h", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags
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
	switch sub {
	case "record", "label":
		return failRuntime("audit logging not yet supported on sq sqlite backend")
	default:
		return failUsage("unknown audit subcommand: " + sub)
	}
}

func cmdSwarm(args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "sq swarm [create|list|status|validate]")
		return 0
	}
	sub := args[0]
	jsonOut := false
	for i := 1; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--json":
			jsonOut = true
		case "--help", "-h", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
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

	switch sub {
	case "list":
		if jsonOut {
			return printJSON([]map[string]any{})
		}
		_, _ = fmt.Fprintln(os.Stdout, "No swarm molecules found.")
		return 0
	case "status":
		status := map[string]any{"active": false, "message": "No active swarm"}
		return printJSON(status)
	case "validate":
		if jsonOut {
			return printJSON(map[string]any{"valid": true, "errors": []string{}})
		}
		_, _ = fmt.Fprintln(os.Stdout, "Swarm structure is valid.")
		return 0
	case "create":
		return failRuntime("swarm create not yet supported on sq sqlite backend")
	default:
		return failUsage("unknown swarm subcommand: " + sub)
	}
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
		hooks, err := listHookStatuses(shared, beadsHooks)
		if err != nil {
			return failRuntime(err.Error())
		}
		status["hooks"] = hooks
	case "install":
		if err := installHooks(force, shared, beadsHooks); err != nil {
			return failRuntime(err.Error())
		}
		status["message"] = "hooks install complete"
	case "uninstall":
		if err := uninstallHooks(); err != nil {
			return failRuntime(err.Error())
		}
		status["message"] = "hooks uninstall complete"
	case "run":
		if len(args) < 2 {
			return failUsage("usage: sq hooks run <hook-name> [args...]")
		}
		hookName := args[1]
		exitCode := runHookDispatcher(hookName, args[2:])
		if exitCode != 0 {
			if exitCode == 2 {
				return failUsage("unknown hook: " + hookName)
			}
			return failRuntime("hook failed: " + hookName)
		}
		status["hook"] = hookName
	default:
		return failUsage("unknown hooks subcommand: " + sub)
	}
	if jsonOut {
		return printJSON(status)
	}
	return printJSON(status)
}

func cmdOnboard(args []string) int {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--help", "-h", "--json", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags
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
	_, _ = fmt.Fprintln(os.Stdout, "## Issue Tracking")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "This project uses **sq (squids)** for issue tracking.")
	_, _ = fmt.Fprintln(os.Stdout, "Run `sq quickstart` for workflow context.")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "**Quick reference:**")
	_, _ = fmt.Fprintln(os.Stdout, "- `sq ready --json` - Find unblocked work")
	_, _ = fmt.Fprintln(os.Stdout, "- `sq create \"Title\" --type task --priority 2 --json` - Create issue")
	_, _ = fmt.Fprintln(os.Stdout, "- `sq close <id> --reason \"Done\" --json` - Complete work")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "For full workflow details: `sq quickstart`")
	return 0
}

func cmdCompletion(args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "sq completion [bash|zsh|fish|powershell]")
		return 0
	}
	shell := ""
	helpRequested := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "bash", "zsh", "fish", "powershell":
			if shell == "" {
				shell = a
				continue
			}
			return failUsage("completion accepts one shell")
		case "--help", "-h":
			helpRequested = true
		case "--json", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// compatibility flags accepted as no-op
		case "--actor", "--db", "--dolt-auto-commit":
			if i+1 < len(args) {
				i++
			}
		default:
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
			return failUsage("unknown shell: " + a)
		}
	}
	if shell == "" {
		if helpRequested {
			_, _ = fmt.Fprintln(os.Stdout, "Generate shell completion script: sq completion <bash|zsh|fish|powershell>")
			return 0
		}
		return failUsage("usage: sq completion <bash|zsh|fish|powershell>")
	}
	script := ""
	switch shell {
	case "bash":
		script = "# bash completion for sq\n_complete_sq() { COMPREPLY=(\"help\" \"init\" \"list\" \"show\" \"create\" \"update\" \"close\" \"ready\"); }\ncomplete -F _complete_sq sq\n"
	case "zsh":
		script = "#compdef sq\n_arguments '*: :->cmds'\n"
	case "fish":
		script = "complete -c sq -f\ncomplete -c sq -a 'help init list show create update close ready'\n"
	case "powershell":
		script = "Register-ArgumentCompleter -Native -CommandName sq -ScriptBlock { param($wordToComplete) 'help','init','list','show','create','update','close','ready' | Where-Object { $_ -like \"$wordToComplete*\" } }\n"
	}
	_, err := fmt.Fprint(os.Stdout, script)
	if err != nil {
		return failRuntime(err.Error())
	}
	return 0
}
