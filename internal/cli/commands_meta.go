package cli

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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

func printDepAddHelp() {
	fmt.Println("Add a dependency between two issues.")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  sq dep add <issue-id> <depends-on-id> [flags]")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  -h, --help   help for add")
	fmt.Println("      --json   Output in JSON format")
}

func printDepRemoveHelp() {
	fmt.Println("Remove a dependency between two issues.")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  sq dep remove <issue-id> <depends-on-id> [flags]")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  -h, --help   help for remove")
	fmt.Println("      --json   Output in JSON format")
}

func printDepListHelp() {
	fmt.Println("List dependencies for an issue.")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  sq dep list <issue-id> [flags]")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  -h, --help   help for list")
	fmt.Println("      --json   Output in JSON format")
}

func cmdDep(args []string) int {
	if len(args) == 0 {
		return failUsage("dep subcommand required")
	}
	sub := args[0]

	switch sub {
	case "add":
		if len(args) >= 2 && (args[1] == "--help" || args[1] == "-h") {
			printDepAddHelp()
			return 0
		}
		if len(args) < 3 {
			return failUsage("usage: sq dep add <issue-id> <depends-on-id> [--json]")
		}
		for _, a := range args[3:] {
			if a == "--json" {
				continue
			}
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
			return failUsage("dep add accepts exactly two positional arguments")
		}
		db, _, err := openTaskDB()
		if err != nil {
			return failRuntime(err.Error())
		}
		defer db.Close()
		if err := store.AddDependency(db, args[1], args[2], "blocks"); err != nil {
			return failRuntime(err.Error())
		}
		return printJSON(map[string]any{"issue_id": args[1], "depends_on_id": args[2], "type": "blocks"})
	case "remove", "rm":
		if len(args) >= 2 && (args[1] == "--help" || args[1] == "-h") {
			printDepRemoveHelp()
			return 0
		}
		if len(args) < 3 {
			return failUsage("usage: sq dep remove <issue-id> <depends-on-id> [--json]")
		}
		for _, a := range args[3:] {
			if a == "--json" {
				continue
			}
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
			return failUsage("dep remove accepts exactly two positional arguments")
		}
		db, _, err := openTaskDB()
		if err != nil {
			return failRuntime(err.Error())
		}
		defer db.Close()
		if err := store.RemoveDependency(db, args[1], args[2]); err != nil {
			return failRuntime(err.Error())
		}
		return printJSON(map[string]any{"issue_id": args[1], "depends_on_id": args[2], "removed": true})
	case "list":
		if len(args) >= 2 && (args[1] == "--help" || args[1] == "-h") {
			printDepListHelp()
			return 0
		}
		if len(args) < 2 {
			return failUsage("usage: sq dep list <issue-id> [--json]")
		}
		for _, a := range args[2:] {
			if a == "--json" {
				continue
			}
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
			return failUsage("dep list accepts exactly one positional argument")
		}
		db, _, err := openTaskDB()
		if err != nil {
			return failRuntime(err.Error())
		}
		defer db.Close()
		deps, err := store.ListDependencies(db, args[1])
		if err != nil {
			return failRuntime(err.Error())
		}
		return printJSON(deps)
	default:
		return failUsage("unknown dep subcommand: " + sub)
	}
}

func printCommentsHelp() {
	fmt.Println("View or manage comments on an issue.")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  # List all comments on an issue")
	fmt.Println("  sq comments bd-123")
	fmt.Println("")
	fmt.Println("  # List comments in JSON format")
	fmt.Println("  sq comments bd-123 --json")
	fmt.Println("")
	fmt.Println("  # Add a comment")
	fmt.Println("  sq comments add bd-123 \"This is a comment\"")
	fmt.Println("")
	fmt.Println("  # Add a comment from a file")
	fmt.Println("  sq comments add bd-123 -f notes.txt")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  sq comments [issue-id] [flags]")
	fmt.Println("  sq comments [command]")
	fmt.Println("")
	fmt.Println("Available Commands:")
	fmt.Println("  add         Add a comment to an issue")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  -h, --help         help for comments")
	fmt.Println("      --local-time   Show timestamps in local time instead of UTC")
	fmt.Println("")
	printGlobalFlags()
	fmt.Println("")
	fmt.Println("Use \"sq comments [command] --help\" for more information about a command.")
}

func printCommentsAddHelp() {
	fmt.Println("Add a comment to an issue.")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  # Add a comment")
	fmt.Println("  sq comments add bd-123 \"Working on this now\"")
	fmt.Println("")
	fmt.Println("  # Add a comment from a file")
	fmt.Println("  sq comments add bd-123 -f notes.txt")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  sq comments add [issue-id] [text] [flags]")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  -a, --author string   Add author to comment")
	fmt.Println("  -f, --file string     Read comment text from file")
	fmt.Println("  -h, --help            help for add")
	fmt.Println("")
	printGlobalFlags()
}

func formatCommentTime(createdAt string, local bool) string {
	if !local {
		return createdAt
	}
	t, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return createdAt
	}
	return t.Local().Format(time.RFC3339)
}

func cmdComments(args []string) int {
	if len(args) == 0 {
		return failUsage("usage: sq comments <issue-id> [flags]")
	}
	if args[0] == "--help" || args[0] == "-h" {
		printCommentsHelp()
		return 0
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()

	if args[0] == "add" {
		jsonOut := false
		issueID := ""
		body := ""
		bodyFile := ""
		author := strings.TrimSpace(os.Getenv("BD_ACTOR"))
		for i := 1; i < len(args); i++ {
			a := args[i]
			switch a {
			case "--json", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
				if a == "--json" {
					jsonOut = true
				}
			case "--help", "-h":
				printCommentsAddHelp()
				return 0
			case "--author", "-a":
				if i+1 >= len(args) {
					return failUsage("missing value for " + a)
				}
				author = args[i+1]
				i++
			case "--file", "-f":
				if i+1 >= len(args) {
					return failUsage("missing value for " + a)
				}
				bodyFile = args[i+1]
				i++
			case "--actor", "--db", "--dolt-auto-commit":
				if i+1 >= len(args) {
					return failUsage("missing value for " + a)
				}
				i++
			default:
				if strings.HasPrefix(a, "-") {
					return failUsage("unknown flag: " + a)
				}
				if issueID == "" {
					issueID = a
				} else if body == "" {
					body = a
				} else {
					return failUsage("usage: sq comments add [issue-id] [text] [flags]")
				}
			}
		}
		if issueID == "" {
			return failUsage("usage: sq comments add [issue-id] [text] [flags]")
		}
		if bodyFile != "" {
			content, err := os.ReadFile(bodyFile)
			if err != nil {
				return failRuntime(err.Error())
			}
			body = string(content)
		}
		if strings.TrimSpace(body) == "" {
			return failUsage("usage: sq comments add [issue-id] [text] [flags]")
		}
		c, err := store.AddComment(db, issueID, author, body)
		if err != nil {
			return failRuntime(err.Error())
		}
		if jsonOut {
			return printJSON(c)
		}
		_, _ = fmt.Fprintf(os.Stdout, "Added comment %d to %s\n", c.ID, issueID)
		return 0
	}

	jsonOut := false
	localTime := false
	issueID := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--json", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			if a == "--json" {
				jsonOut = true
			}
		case "--local-time":
			localTime = true
		case "--actor", "--db", "--dolt-auto-commit":
			if i+1 >= len(args) {
				return failUsage("missing value for " + a)
			}
			i++
		default:
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
			if issueID == "" {
				issueID = a
			} else {
				return failUsage("usage: sq comments <issue-id> [flags]")
			}
		}
	}
	if issueID == "" {
		return failUsage("usage: sq comments <issue-id> [flags]")
	}
	comments, err := store.ListComments(db, issueID)
	if err != nil {
		return failRuntime(err.Error())
	}
	if jsonOut {
		return printJSON(comments)
	}
	if len(comments) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No comments found")
		return 0
	}
	for _, c := range comments {
		author := c.Author
		if strings.TrimSpace(author) == "" {
			author = "unknown"
		}
		_, _ = fmt.Fprintf(os.Stdout, "%d  %s  %s\n", c.ID, formatCommentTime(c.CreatedAt, localTime), author)
		_, _ = fmt.Fprintf(os.Stdout, "    %s\n", c.Body)
	}
	return 0
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

func printChildrenHelp() {
	fmt.Println("List all beads that are children of the specified parent bead.")
	fmt.Println("")
	fmt.Println("This is a convenience alias for 'sq list --parent <id>'.")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  sq children bd-abc123        # List children of bd-abc123")
	fmt.Println("  sq children bd-abc123 --json # List children in JSON format")
	fmt.Println("  sq children bd-abc123 --pretty # Show children in tree format")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  sq children <parent-id> [flags]")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  -h, --help   help for children")
	fmt.Println("      --pretty Display children in a tree format with status/priority symbols")
	fmt.Println("")
	printGlobalFlags()
}

func cmdChildren(args []string) int {
	jsonOut := false
	pretty := false
	parentID := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--json":
			jsonOut = true
		case "--pretty":
			pretty = true
		case "--help", "-h":
			printChildrenHelp()
			return 0
		case "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags
		case "--actor", "--db", "--dolt-auto-commit", "--parent":
			if i+1 >= len(args) {
				return failUsage("missing value for " + a)
			}
			if a == "--parent" {
				parentID = args[i+1]
			}
			i++
		default:
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
			if parentID == "" {
				parentID = a
			} else {
				return failUsage("usage: sq children <parent-id> [flags]")
			}
		}
	}
	if parentID == "" {
		return failUsage("usage: sq children <parent-id> [flags]")
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	items, err := store.ListChildren(db, parentID)
	if err != nil {
		return failRuntime(err.Error())
	}
	if jsonOut {
		return printJSON(items)
	}
	if pretty || !jsonOut {
		if len(items) == 0 {
			_, _ = fmt.Fprintln(os.Stdout, "No children found")
			return 0
		}
		for _, item := range items {
			_, _ = fmt.Fprintf(os.Stdout, "├── %s [%s P%d] %s\n", item.ID, item.Status, item.Priority, item.Title)
		}
		return 0
	}
	return printJSON(items)
}

func printBlockedHelp() {
	fmt.Println("Show blocked issues")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  sq blocked [flags]")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  -h, --help            help for blocked")
	fmt.Println("      --parent string   Filter to descendants of this bead/epic")
	fmt.Println("")
	printGlobalFlags()
}

func cmdBlocked(args []string) int {
	jsonOut := false
	parentID := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOut = true
		case "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			continue
		case "--help", "-h":
			printBlockedHelp()
			return 0
		case "--parent":
			if i+1 >= len(args) {
				return failUsage("missing value for --parent")
			}
			parentID = args[i+1]
			i++
		case "--actor", "--db", "--dolt-auto-commit":
			if i+1 >= len(args) {
				return failUsage("missing value for " + args[i])
			}
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				return failUsage("unknown flag: " + args[i])
			}
			return failUsage("blocked does not accept positional arguments")
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
	if parentID != "" {
		descendants, err := blockedDescendants(db, parentID)
		if err != nil {
			return failRuntime(err.Error())
		}
		filtered := make([]store.BlockedItem, 0, len(items))
		for _, item := range items {
			if descendants[item.ID] {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if jsonOut {
		return printJSON(items)
	}
	if len(items) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No blocked issues found")
		return 0
	}
	_, _ = fmt.Fprintf(os.Stdout, "Found %d blocked issue(s):\n", len(items))
	for _, item := range items {
		_, _ = fmt.Fprintf(os.Stdout, "- %s [%s] %s (blocked by %d: %s)\n", item.ID, item.Status, item.Title, item.BlockedByCount, strings.Join(item.BlockedBy, ", "))
	}
	return 0
}

func blockedDescendants(db *sql.DB, parentID string) (map[string]bool, error) {
	seen := map[string]bool{}
	queue := []string{parentID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		children, err := store.ListChildren(db, current)
		if err != nil {
			return nil, err
		}
		for _, child := range children {
			if seen[child.ID] {
				continue
			}
			seen[child.ID] = true
			queue = append(queue, child.ID)
		}
	}
	return seen, nil
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

func printTypesHelp() {
	_, _ = fmt.Fprintln(os.Stdout, "List the built-in sq issue types.")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Usage:")
	_, _ = fmt.Fprintln(os.Stdout, "  sq types [flags]")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Flags:")
	_, _ = fmt.Fprintln(os.Stdout, "  -h, --help   help for types")
	_, _ = fmt.Fprintln(os.Stdout, "      --json   Output in JSON format")
	_, _ = fmt.Fprintln(os.Stdout, "")
	printGlobalFlags()
}

func cmdTypes(args []string) int {
	jsonOut := false
	coreTypes := []map[string]string{
		{"name": "task", "description": "General work item (default)"},
		{"name": "bug", "description": "Bug report or defect"},
		{"name": "feature", "description": "New feature or enhancement"},
		{"name": "chore", "description": "Maintenance or housekeeping"},
		{"name": "epic", "description": "Large body of work spanning multiple issues"},
		{"name": "decision", "description": "Architecture decision record (ADR)"},
	}
	for _, a := range args {
		switch a {
		case "--json":
			jsonOut = true
		case "--help", "-h":
			printTypesHelp()
			return 0
		default:
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
			return failUsage("types does not accept positional arguments")
		}
	}
	if jsonOut {
		return printJSON(map[string]any{"core_types": coreTypes})
	}
	_, _ = fmt.Fprintln(os.Stdout, "Core issue types:")
	for _, item := range coreTypes {
		_, _ = fmt.Fprintf(os.Stdout, "- %s: %s\n", item["name"], item["description"])
	}
	return 0
}
