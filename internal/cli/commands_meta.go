package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/berdon/squids/internal/store"
)

func printLabelHelp() {
	fmt.Println("Manage labels on tasks")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  sq label add <id> <label> [--json]")
	fmt.Println("  sq label remove <id> <label> [--json]")
	fmt.Println("  sq label list <id> [--json]")
	fmt.Println("  sq label list-all [--json]")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  --json   output JSON")
}

func hasJSONFlag(args []string) bool {
	for _, a := range args {
		if a == "--json" {
			return true
		}
	}
	return false
}

func cmdLabel(args []string) int {
	if len(args) == 0 {
		return failUsage("label subcommand required")
	}
	sub := args[0]
	if sub == "--help" || sub == "-h" {
		printLabelHelp()
		return 0
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()

	switch sub {
	case "add":
		if len(args) >= 2 && (args[1] == "--help" || args[1] == "-h") {
			fmt.Println("Usage:")
			fmt.Println("  sq label add <id> <label> [--json]")
			fmt.Println("")
			fmt.Println("Flags:")
			fmt.Println("  --json   output JSON")
			return 0
		}
		if len(args) < 3 {
			return failUsage("usage: sq label add <id> <label> [--json]")
		}
		t, err := store.AddLabel(db, args[1], args[2])
		if err != nil {
			return failRuntime(err.Error())
		}
		if hasJSONFlag(args[3:]) {
			return printJSON(t)
		}
		_, _ = fmt.Fprintf(os.Stdout, "✓ Added label '%s' to %s\n", args[2], args[1])
		return 0
	case "remove":
		if len(args) >= 2 && (args[1] == "--help" || args[1] == "-h") {
			fmt.Println("Usage:")
			fmt.Println("  sq label remove <id> <label> [--json]")
			fmt.Println("")
			fmt.Println("Flags:")
			fmt.Println("  --json   output JSON")
			return 0
		}
		if len(args) < 3 {
			return failUsage("usage: sq label remove <id> <label> [--json]")
		}
		t, err := store.RemoveLabel(db, args[1], args[2])
		if err != nil {
			return failRuntime(err.Error())
		}
		if hasJSONFlag(args[3:]) {
			return printJSON(t)
		}
		_, _ = fmt.Fprintf(os.Stdout, "✓ Removed label '%s' from %s\n", args[2], args[1])
		return 0
	case "list":
		if len(args) >= 2 && (args[1] == "--help" || args[1] == "-h") {
			fmt.Println("Usage:")
			fmt.Println("  sq label list <id> [--json]")
			fmt.Println("")
			fmt.Println("Flags:")
			fmt.Println("  --json   output JSON")
			return 0
		}
		if len(args) < 2 {
			return failUsage("usage: sq label list <id> [--json]")
		}
		labels, err := store.ListLabels(db, args[1])
		if err != nil {
			return failRuntime(err.Error())
		}
		if hasJSONFlag(args[2:]) {
			return printJSON(labels)
		}
		_, _ = fmt.Fprintf(os.Stdout, "\n🏷 Labels for %s:\n", args[1])
		if len(labels) == 0 {
			_, _ = fmt.Fprintln(os.Stdout, "  (none)")
		} else {
			for _, l := range labels {
				_, _ = fmt.Fprintf(os.Stdout, "  - %s\n", l)
			}
		}
		_, _ = fmt.Fprintln(os.Stdout)
		return 0
	case "list-all":
		labels, err := store.ListAllLabels(db)
		if err != nil {
			return failRuntime(err.Error())
		}
		if hasJSONFlag(args[1:]) {
			return printJSON(labels)
		}
		tasks, err := store.ListTasks(db)
		if err != nil {
			return failRuntime(err.Error())
		}
		counts := map[string]int{}
		for _, t := range tasks {
			for _, l := range t.Labels {
				counts[l]++
			}
		}
		_, _ = fmt.Fprintf(os.Stdout, "\n🏷 All labels (%d unique):\n", len(labels))
		for _, l := range labels {
			_, _ = fmt.Fprintf(os.Stdout, "  %-16s (%d issues)\n", l, counts[l])
		}
		_, _ = fmt.Fprintln(os.Stdout)
		return 0
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

func cmdDefer(args []string) int {
	if len(args) == 0 {
		return failUsage("usage: sq defer <id> [<id>...] [--json]")
	}
	ids := make([]string, 0)
	for _, a := range args {
		if a == "--json" {
			continue
		}
		if strings.HasPrefix(a, "-") {
			return failUsage("unknown flag: " + a)
		}
		ids = append(ids, a)
	}
	if len(ids) == 0 {
		return failUsage("at least one id is required")
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	out := make([]*store.Task, 0, len(ids))
	for _, id := range ids {
		status := "deferred"
		t, err := store.UpdateTask(db, id, store.UpdateInput{Status: &status})
		if err != nil {
			return failRuntime(err.Error())
		}
		out = append(out, t)
	}
	return printJSON(out)
}

func cmdUndefer(args []string) int {
	if len(args) == 0 {
		return failUsage("usage: sq undefer <id> [<id>...] [--json]")
	}
	ids := make([]string, 0)
	for _, a := range args {
		if a == "--json" {
			continue
		}
		if strings.HasPrefix(a, "-") {
			return failUsage("unknown flag: " + a)
		}
		ids = append(ids, a)
	}
	if len(ids) == 0 {
		return failUsage("at least one id is required")
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	out := make([]*store.Task, 0, len(ids))
	for _, id := range ids {
		status := "open"
		t, err := store.UpdateTask(db, id, store.UpdateInput{Status: &status})
		if err != nil {
			return failRuntime(err.Error())
		}
		out = append(out, t)
	}
	return printJSON(out)
}

func cmdSetState(args []string) int {
	if len(args) < 2 {
		return failUsage("usage: sq set-state <issue-id> <dimension>=<value> [--reason TEXT] [--json]")
	}
	for i := 2; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--reason", "--actor", "--db", "--dolt-auto-commit":
			if i+1 < len(args) {
				i++
			}
		case "--json", "--help", "-h", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags
		default:
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
		}
	}
	return failRuntime("set-state compatibility surface only; state events not yet supported on sq sqlite backend")
}

func cmdRename(args []string) int {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			_, _ = fmt.Fprintln(os.Stdout, "Rename issue IDs while preserving relationships.")
			_, _ = fmt.Fprintln(os.Stdout, "Usage: sq rename <old-id> <new-id> [--json]")
			return 0
		}
	}
	if len(args) < 2 {
		return failUsage("usage: sq rename <old-id> <new-id> [--json]")
	}
	oldID, newID := args[0], args[1]
	for i := 2; i < len(args); i++ {
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
	t, err := store.RenameTask(db, oldID, newID)
	if err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(t)
}

func cmdRenamePrefix(args []string) int {
	if len(args) < 1 {
		return failUsage("usage: sq rename-prefix <new-prefix> [--json] OR sq rename-prefix <old-prefix> <new-prefix> [--json]")
	}
	filtered := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--json" {
			continue
		}
		if strings.HasPrefix(a, "-") {
			return failUsage("unknown flag: " + a)
		}
		filtered = append(filtered, a)
	}
	if len(filtered) < 1 || len(filtered) > 2 {
		return failUsage("usage: sq rename-prefix <new-prefix> [--json] OR sq rename-prefix <old-prefix> <new-prefix> [--json]")
	}
	oldPrefix := "bd"
	newPrefix := filtered[0]
	if len(filtered) == 2 {
		oldPrefix = filtered[0]
		newPrefix = filtered[1]
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	n, err := store.RenamePrefix(db, oldPrefix, newPrefix)
	if err != nil {
		return failRuntime(err.Error())
	}
	return printJSON(map[string]any{"renamed": n, "old_prefix": oldPrefix, "new_prefix": newPrefix})
}

func cmdDuplicate(args []string) int {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			_, _ = fmt.Fprintln(os.Stdout, "Mark issue as duplicate of canonical issue.")
			_, _ = fmt.Fprintln(os.Stdout, "Usage: sq duplicate <id> --of <canonical-id> [--json]")
			return 0
		}
	}
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
