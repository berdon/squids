package cli

import (
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/berdon/squids/internal/store"
)

func printInitHelp() {
	_, _ = fmt.Fprintln(os.Stdout, "Initialize sq task storage in the current workspace.")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Usage:")
	_, _ = fmt.Fprintln(os.Stdout, "  sq init [flags]")
	_, _ = fmt.Fprintln(os.Stdout, "")
	printGlobalFlags()
}

func cmdInit(args []string) int {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--help", "-h":
			printInitHelp()
			return 0
		case "--json", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags (no-op)
		case "--actor", "--db", "--dolt-auto-commit", "--prefix":
			if i+1 >= len(args) {
				return failUsage("missing value for " + a)
			}
			i++
		default:
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
			return failUsage("init does not accept positional arguments")
		}
	}

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

func printReadyHelp() {
	_, _ = fmt.Fprintln(os.Stdout, "Show ready work (open issues with no active blockers).")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Usage:")
	_, _ = fmt.Fprintln(os.Stdout, "  sq ready [flags]")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Flags:")
	_, _ = fmt.Fprintln(os.Stdout, "  -a, --assignee string     Filter by assignee")
	_, _ = fmt.Fprintln(os.Stdout, "      --gated               Accepted for bd parity (no-op)")
	_, _ = fmt.Fprintln(os.Stdout, "      --has-metadata-key    Filter issues that have this metadata key set")
	_, _ = fmt.Fprintln(os.Stdout, "  -h, --help               help for ready")
	_, _ = fmt.Fprintln(os.Stdout, "      --include-deferred    Accepted for bd parity (no-op)")
	_, _ = fmt.Fprintln(os.Stdout, "      --include-ephemeral   Accepted for bd parity (no-op)")
	_, _ = fmt.Fprintln(os.Stdout, "  -l, --label strings      Filter by labels (AND: must have ALL)")
	_, _ = fmt.Fprintln(os.Stdout, "      --label-any strings   Filter by labels (OR: must have AT LEAST ONE)")
	_, _ = fmt.Fprintln(os.Stdout, "  -n, --limit int          Maximum issues to show")
	_, _ = fmt.Fprintln(os.Stdout, "      --metadata-field      Filter by metadata field (key=value, repeatable)")
	_, _ = fmt.Fprintln(os.Stdout, "      --mol string          Accepted for bd parity (no-op)")
	_, _ = fmt.Fprintln(os.Stdout, "      --mol-type string     Accepted for bd parity (no-op)")
	_, _ = fmt.Fprintln(os.Stdout, "      --parent string       Filter to descendants of this parent issue")
	_, _ = fmt.Fprintln(os.Stdout, "      --plain               Accepted for bd parity (plain text is the default)")
	_, _ = fmt.Fprintln(os.Stdout, "      --pretty              Accepted for bd parity (plain text is the default)")
	_, _ = fmt.Fprintln(os.Stdout, "  -p, --priority int       Filter by priority")
	_, _ = fmt.Fprintln(os.Stdout, "      --rig string          Accepted for bd parity (no-op)")
	_, _ = fmt.Fprintln(os.Stdout, "  -s, --sort string        Sort policy: priority (default), oldest")
	_, _ = fmt.Fprintln(os.Stdout, "  -t, --type string        Filter by issue type")
	_, _ = fmt.Fprintln(os.Stdout, "  -u, --unassigned         Show only unassigned issues")
	_, _ = fmt.Fprintln(os.Stdout, "")
	printGlobalFlags()
}

func cmdReady(args []string) int {
	var (
		assignee       *string
		hasMetadataKey string
		jsonOut        bool
		labelsAll      []string
		labelsAny      []string
		limit          int
		metadataFields []string
		parent         string
		priority       *int
		sortBy         = "priority"
		typeFilter     string
		unassigned     bool
	)

	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--help", "-h":
			printReadyHelp()
			return 0
		case "--json":
			jsonOut = true
		case "--gated", "--include-deferred", "--include-ephemeral", "--plain", "--pretty", "--quiet", "-q", "--verbose", "-v", "--profile", "--readonly", "--sandbox":
			// accepted compatibility flags (no-op)
		case "--actor", "--db", "--dolt-auto-commit", "--mol", "--mol-type", "--rig":
			if i+1 >= len(args) {
				return failUsage("missing value for " + a)
			}
			i++
		case "--assignee", "-a":
			if i+1 >= len(args) {
				return failUsage("missing value for " + a)
			}
			v := args[i+1]
			assignee = &v
			i++
		case "--has-metadata-key":
			if i+1 >= len(args) {
				return failUsage("missing value for " + a)
			}
			hasMetadataKey = args[i+1]
			i++
		case "--label", "-l":
			if i+1 >= len(args) {
				return failUsage("missing value for " + a)
			}
			labelsAll = append(labelsAll, splitCSV(args[i+1])...)
			i++
		case "--label-any":
			if i+1 >= len(args) {
				return failUsage("missing value for " + a)
			}
			labelsAny = append(labelsAny, splitCSV(args[i+1])...)
			i++
		case "--limit", "-n":
			if i+1 >= len(args) {
				return failUsage("missing value for " + a)
			}
			v, err := strconv.Atoi(args[i+1])
			if err != nil || v < 0 {
				return failUsage("invalid --limit")
			}
			limit = v
			i++
		case "--metadata-field":
			if i+1 >= len(args) {
				return failUsage("missing value for " + a)
			}
			metadataFields = append(metadataFields, args[i+1])
			i++
		case "--parent":
			if i+1 >= len(args) {
				return failUsage("missing value for " + a)
			}
			parent = args[i+1]
			i++
		case "--priority", "-p":
			if i+1 >= len(args) {
				return failUsage("missing value for " + a)
			}
			v, err := strconv.Atoi(args[i+1])
			if err != nil {
				return failUsage("invalid --priority")
			}
			priority = &v
			i++
		case "--sort", "-s":
			if i+1 >= len(args) {
				return failUsage("missing value for " + a)
			}
			sortBy = args[i+1]
			i++
		case "--type", "-t":
			if i+1 >= len(args) {
				return failUsage("missing value for " + a)
			}
			typeFilter = args[i+1]
			i++
		case "--unassigned", "-u":
			unassigned = true
		default:
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
			return failUsage("ready does not accept positional arguments")
		}
	}

	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	items, err := store.ReadyTasks(db)
	if err != nil {
		return failRuntime(err.Error())
	}

	filtered := make([]store.Task, 0, len(items))
	for _, t := range items {
		if assignee != nil && t.Assignee != *assignee {
			continue
		}
		if unassigned && strings.TrimSpace(t.Assignee) != "" {
			continue
		}
		if priority != nil && t.Priority != *priority {
			continue
		}
		if typeFilter != "" && t.IssueType != typeFilter {
			continue
		}
		if parent != "" && !containsString(t.Deps, parent) {
			continue
		}
		if hasMetadataKey != "" {
			if _, ok := t.Metadata[hasMetadataKey]; !ok {
				continue
			}
		}
		if !matchesAllLabels(t.Labels, labelsAll) || !matchesAnyLabel(t.Labels, labelsAny) {
			continue
		}
		if !matchesMetadataFields(t.Metadata, metadataFields) {
			continue
		}
		filtered = append(filtered, t)
	}

	sortReadyTasks(filtered, sortBy)
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	if jsonOut {
		return printJSON(filtered)
	}
	return printTaskCollection(db, filtered, "Found %d ready issue(s):\n", "No ready issues.\n")
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func matchesAllLabels(taskLabels, required []string) bool {
	if len(required) == 0 {
		return true
	}
	set := map[string]struct{}{}
	for _, label := range taskLabels {
		set[label] = struct{}{}
	}
	for _, label := range required {
		if _, ok := set[label]; !ok {
			return false
		}
	}
	return true
}

func matchesAnyLabel(taskLabels, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	set := map[string]struct{}{}
	for _, label := range taskLabels {
		set[label] = struct{}{}
	}
	for _, label := range allowed {
		if _, ok := set[label]; ok {
			return true
		}
	}
	return false
}

func matchesMetadataFields(metadata map[string]string, fields []string) bool {
	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 {
			return false
		}
		if metadata[parts[0]] != parts[1] {
			return false
		}
	}
	return true
}

func sortReadyTasks(tasks []store.Task, sortBy string) {
	switch sortBy {
	case "priority", "", "hybrid":
		sort.SliceStable(tasks, func(i, j int) bool {
			if tasks[i].Priority != tasks[j].Priority {
				return tasks[i].Priority < tasks[j].Priority
			}
			return tasks[i].CreatedAt < tasks[j].CreatedAt
		})
	case "oldest":
		sort.SliceStable(tasks, func(i, j int) bool {
			return tasks[i].CreatedAt < tasks[j].CreatedAt
		})
	default:
		// preserve store order for unknown values to keep compatibility forgiving
	}
}

func printTaskCollection(db *sql.DB, tasks []store.Task, headingFmt, emptyMsg string) int {
	if len(tasks) == 0 {
		_, _ = fmt.Fprint(os.Stdout, emptyMsg)
		return 0
	}

	allTasks, err := store.ListTasks(db)
	if err != nil {
		return failRuntime(err.Error())
	}
	parentByChild, err := parentLinks(db)
	if err != nil {
		return failRuntime(err.Error())
	}

	allByID := make(map[string]store.Task, len(allTasks))
	order := make(map[string]int, len(allTasks))
	for i, task := range allTasks {
		allByID[task.ID] = task
		order[task.ID] = i
	}

	visible := make(map[string]store.Task, len(tasks))
	for _, task := range tasks {
		visible[task.ID] = task
		seen := map[string]struct{}{task.ID: {}}
		for parentID := parentByChild[task.ID]; parentID != ""; parentID = parentByChild[parentID] {
			if _, ok := seen[parentID]; ok {
				break
			}
			seen[parentID] = struct{}{}
			if parent, ok := allByID[parentID]; ok {
				visible[parentID] = parent
			}
		}
	}

	childrenByParent := map[string][]string{}
	roots := make([]string, 0, len(visible))
	for id := range visible {
		parentID := parentByChild[id]
		if parentID != "" {
			if _, ok := visible[parentID]; ok {
				childrenByParent[parentID] = append(childrenByParent[parentID], id)
				continue
			}
		}
		roots = append(roots, id)
	}

	sortIDsByOrder := func(ids []string) {
		sort.SliceStable(ids, func(i, j int) bool {
			return order[ids[i]] < order[ids[j]]
		})
	}
	sortIDsByOrder(roots)
	for parentID := range childrenByParent {
		sortIDsByOrder(childrenByParent[parentID])
	}

	_, _ = fmt.Fprintf(os.Stdout, headingFmt, len(tasks))
	for i, id := range roots {
		last := i == len(roots)-1
		printTaskTree(id, visible, childrenByParent, "", last, true)
	}
	return 0
}

func printTaskTree(id string, visible map[string]store.Task, childrenByParent map[string][]string, prefix string, last bool, root bool) {
	task, ok := visible[id]
	if !ok {
		return
	}
	linePrefix := prefix
	if !root {
		connector := "├── "
		if last {
			connector = "└── "
		}
		linePrefix += connector
	}
	_, _ = fmt.Fprintf(os.Stdout, "%s%s\n", linePrefix, humanTaskSummary(task))

	children := childrenByParent[id]
	nextPrefix := prefix
	if !root {
		if last {
			nextPrefix += "    "
		} else {
			nextPrefix += "│   "
		}
	}
	for i, childID := range children {
		printTaskTree(childID, visible, childrenByParent, nextPrefix, i == len(children)-1, false)
	}
}

func humanTaskSummary(t store.Task) string {
	statusIcon := "○"
	statusBadge := "●"
	if strings.EqualFold(t.Status, "closed") {
		statusIcon = "✓"
		statusBadge = "✓"
	}
	assignee := ""
	if strings.TrimSpace(t.Assignee) != "" {
		assignee = " @" + t.Assignee
	}
	issueType := t.IssueType
	if issueType == "" {
		issueType = "task"
	}
	return fmt.Sprintf("%s %s [%s P%d] [%s]%s - %s", statusIcon, t.ID, statusBadge, t.Priority, issueType, assignee, t.Title)
}

func parentLinks(db *sql.DB) (map[string]string, error) {
	rows, err := db.Query(`SELECT issue_id, depends_on_id FROM dependencies WHERE dep_type='parent-child' ORDER BY depends_on_id, issue_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := map[string]string{}
	for rows.Next() {
		var issueID, parentID string
		if err := rows.Scan(&issueID, &parentID); err != nil {
			return nil, err
		}
		links[issueID] = parentID
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return links, nil
}

func cmdCreate(args []string) int {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			printCreateHelp()
			return 0
		}
	}
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

func cmdQ(args []string) int {
	if len(args) == 0 {
		return failUsage("title is required")
	}
	creator := strings.TrimSpace(os.Getenv("BD_ACTOR"))
	if creator == "" {
		creator = strings.TrimSpace(os.Getenv("USER"))
	}
	in := store.CreateInput{Title: args[0], IssueType: "task", Priority: 2, Creator: creator}
	jsonOut := false
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
			jsonOut = true
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
	if jsonOut {
		return printJSON(map[string]any{"id": t.ID})
	}
	_, _ = fmt.Fprintln(os.Stdout, t.ID)
	return 0
}

func cmdShow(args []string) int {
	if len(args) == 0 {
		return failUsage("id is required")
	}
	id := ""
	jsonOut := false
	for _, a := range args {
		switch a {
		case "--help", "-h":
			_, _ = fmt.Fprintln(os.Stdout, "Show details for a single issue.")
			_, _ = fmt.Fprintln(os.Stdout, "Usage: sq show <id> [--json]")
			return 0
		case "--json":
			jsonOut = true
		default:
			if strings.HasPrefix(a, "-") {
				return failUsage("unknown flag: " + a)
			}
			if id == "" {
				id = a
			}
		}
	}
	if id == "" {
		return failUsage("id is required")
	}
	db, _, err := openTaskDB()
	if err != nil {
		return failRuntime(err.Error())
	}
	defer db.Close()
	t, err := store.ShowTask(db, id)
	if err != nil {
		return failRuntime(err.Error())
	}
	if jsonOut {
		return printJSON(t)
	}
	_, _ = fmt.Fprintf(os.Stdout, "%s [%s P%d] %s\n", t.ID, t.IssueType, t.Priority, t.Title)
	return 0
}

func cmdList(args []string) int {
	jsonOut := false
	for _, a := range args {
		if a == "--json" {
			jsonOut = true
			continue
		}
		if a == "--flat" || a == "--no-pager" {
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
	if jsonOut {
		return printJSON(tasks)
	}
	return printTaskCollection(db, tasks, "Found %d issue(s):\n", "No issues found.\n")
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
