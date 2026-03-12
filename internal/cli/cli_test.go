package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCLI(t *testing.T, dbPath string, args ...string) (code int, stdout string, stderr string) {
	t.Helper()
	oldDB := os.Getenv("SQ_DB_PATH")
	_ = os.Setenv("SQ_DB_PATH", dbPath)
	defer func() { _ = os.Setenv("SQ_DB_PATH", oldDB) }()

	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr
	defer func() {
		os.Stdout, os.Stderr = oldOut, oldErr
	}()

	code = Run(args)
	_ = wOut.Close()
	_ = wErr.Close()
	outB, _ := io.ReadAll(rOut)
	errB, _ := io.ReadAll(rErr)
	return code, string(outB), string(errB)
}

func firstID(t *testing.T, s string) string {
	t.Helper()
	var obj map[string]any
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		t.Fatalf("unmarshal json: %v; payload=%q", err, s)
	}
	id, _ := obj["id"].(string)
	if id == "" {
		t.Fatalf("missing id in payload=%q", s)
	}
	return id
}

func TestRun_RuntimeFailurePath(t *testing.T) {
	// Force DB open failure by making parent a non-directory path.
	tmp := t.TempDir()
	bad := filepath.Join(tmp, "notadir", "tasks.sqlite")
	if err := os.WriteFile(filepath.Join(tmp, "notadir"), []byte("x"), 0o644); err != nil {
		t.Fatalf("seed bad path: %v", err)
	}
	code, _, errOut := runCLI(t, bad, "init", "--json")
	if code != 1 || errOut == "" {
		t.Fatalf("expected runtime failure code=1 err=%q", errOut)
	}
}

func TestRun_HelpAndUnknown(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	code, out, _ := runCLI(t, db, "help")
	if code != 0 || !strings.Contains(out, "Usage:") || !strings.Contains(out, "Global Flags:") {
		t.Fatalf("help failed code=%d out=%q", code, out)
	}
	code, out, _ = runCLI(t, db, "help", "--all")
	if code != 0 || !strings.Contains(out, "# sq — Complete Command Reference") || !strings.Contains(out, "## Table of Contents") {
		t.Fatalf("help --all failed code=%d out=%q", code, out)
	}
	code, out, _ = runCLI(t, db, "help", "--help")
	if code != 0 || !strings.Contains(out, "Flags:") || !strings.Contains(out, "Global Flags:") || !strings.Contains(out, "--all") {
		t.Fatalf("help --help failed code=%d out=%q", code, out)
	}
	code, _, _ = runCLI(t, db, "-h")
	if code != 0 {
		t.Fatalf("-h failed code=%d", code)
	}
	code, _, _ = runCLI(t, db, "--help")
	if code != 0 {
		t.Fatalf("--help failed code=%d", code)
	}

	code, out, _ = runCLI(t, db, "help", "create")
	if code != 0 || !strings.Contains(out, "Usage:") || !strings.Contains(out, "help for create") || !strings.Contains(out, "Global Flags:") {
		t.Fatalf("help create failed code=%d out=%q", code, out)
	}

	code, out, _ = runCLI(t, db, "help", "label")
	if code != 0 || !strings.Contains(out, "Usage:") || !strings.Contains(out, "Flags:") {
		t.Fatalf("help label failed code=%d out=%q", code, out)
	}

	code, out, _ = runCLI(t, db, "help", "children")
	if code != 0 || !strings.Contains(out, "sq children <parent-id> [flags]") || !strings.Contains(out, "--pretty") || !strings.Contains(out, "Global Flags:") {
		t.Fatalf("help children failed code=%d out=%q", code, out)
	}

	code, out, _ = runCLI(t, db, "help", "comments")
	if code != 0 || !strings.Contains(out, "sq comments [issue-id] [flags]") || !strings.Contains(out, "--local-time") {
		t.Fatalf("help comments failed code=%d out=%q", code, out)
	}

	code, out, _ = runCLI(t, db, "label", "--help")
	if code != 0 || !strings.Contains(out, "sq label add") {
		t.Fatalf("label --help failed code=%d out=%q", code, out)
	}
	for _, args := range [][]string{{"label", "add", "--help"}, {"label", "remove", "--help"}, {"label", "list", "--help"}} {
		code, out, _ = runCLI(t, db, args...)
		if code != 0 || !strings.Contains(out, "Usage:") || !strings.Contains(out, "Flags:") {
			t.Fatalf("%v help failed code=%d out=%q", args, code, out)
		}
	}

	code, out, _ = runCLI(t, db, "help", "query")
	if code != 0 || !strings.Contains(out, "Usage:") || !strings.Contains(out, "Flags:") {
		t.Fatalf("help query failed code=%d out=%q", code, out)
	}
	code, out, _ = runCLI(t, db, "query", "--help")
	if code != 0 || !strings.Contains(out, "sq query") {
		t.Fatalf("query --help failed code=%d out=%q", code, out)
	}
	code, out, _ = runCLI(t, db, "show", "--help")
	if code != 0 || !strings.Contains(out, "sq show <id>") || !strings.Contains(out, "--json") {
		t.Fatalf("show --help failed code=%d out=%q", code, out)
	}
	code, out, _ = runCLI(t, db, "help", "ready")
	if code != 0 || !strings.Contains(out, "help for ready") || !strings.Contains(out, "--assignee") {
		t.Fatalf("help ready failed code=%d out=%q", code, out)
	}
	code, out, _ = runCLI(t, db, "ready", "--help")
	if code != 0 || !strings.Contains(out, "Usage:") || !strings.Contains(out, "--unassigned") {
		t.Fatalf("ready --help failed code=%d out=%q", code, out)
	}
	code, out, _ = runCLI(t, db, "help", "gate")
	if code != 0 || !strings.Contains(out, "sq gate list") {
		t.Fatalf("help gate failed code=%d out=%q", code, out)
	}
	code, out, _ = runCLI(t, db, "help", "backup")
	if code != 0 || !strings.Contains(out, "sq backup") {
		t.Fatalf("help backup failed code=%d out=%q", code, out)
	}
	code, out, _ = runCLI(t, db, "help", "quickstart")
	if code != 0 || !strings.Contains(out, "Usage:") || !strings.Contains(out, "Global Flags:") {
		t.Fatalf("help quickstart failed code=%d out=%q", code, out)
	}
	code, out, _ = runCLI(t, db, "gate", "--help")
	if code != 0 || !strings.Contains(out, "Usage:") {
		t.Fatalf("gate --help failed code=%d out=%q", code, out)
	}

	code, _, err := runCLI(t, db, "nope")
	if code != 2 || !strings.Contains(err, "unknown command") {
		t.Fatalf("unknown command failed code=%d err=%q", code, err)
	}
}

func TestListAndReadyNestChildrenUnderEpics(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	if code, _, _ := runCLI(t, db, "init", "--json"); code != 0 {
		t.Fatalf("init failed: %d", code)
	}

	code, out, _ := runCLI(t, db, "create", "Epic", "--type", "epic", "--priority", "1", "--json")
	if code != 0 {
		t.Fatalf("create epic failed: %d", code)
	}
	epicID := firstID(t, out)

	code, out, _ = runCLI(t, db, "create", "Child", "--type", "task", "--priority", "2", "--deps", "parent-child:"+epicID, "--json")
	if code != 0 {
		t.Fatalf("create child failed: %d", code)
	}
	childID := firstID(t, out)

	code, out, _ = runCLI(t, db, "list")
	if code != 0 || !strings.Contains(out, epicID) || !strings.Contains(out, "└── ○ "+childID) {
		t.Fatalf("list nesting failed code=%d out=%q", code, out)
	}
	if strings.Index(out, epicID) > strings.Index(out, childID) {
		t.Fatalf("expected epic before child in list output: %q", out)
	}

	code, out, _ = runCLI(t, db, "ready")
	if code != 0 || !strings.Contains(out, epicID) || !strings.Contains(out, "└── ○ "+childID) {
		t.Fatalf("ready nesting failed code=%d out=%q", code, out)
	}
}

func TestRun_EndToEndCommandFamilies(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")

	if code, _, _ := runCLI(t, db, "init", "--json"); code != 0 {
		t.Fatalf("init failed: %d", code)
	}
	if code, out, _ := runCLI(t, db, "ready", "--json"); code != 0 || strings.TrimSpace(out) != "[]" {
		t.Fatalf("ready failed: code=%d out=%q", code, out)
	}

	code, out, _ := runCLI(t, db, "create", "Task A", "--type", "task", "--priority", "1", "--description", "desc", "--json")
	if code != 0 {
		t.Fatalf("create A failed: %d", code)
	}
	aID := firstID(t, out)

	code, out, _ = runCLI(t, db, "create", "Task B", "--type", "task", "--priority", "2", "--json")
	if code != 0 {
		t.Fatalf("create B failed: %d", code)
	}
	bID := firstID(t, out)

	code, out, _ = runCLI(t, db, "create", "Child", "--type", "task", "--priority", "2", "--deps", "parent-child:"+aID, "--json")
	if code != 0 {
		t.Fatalf("create child failed: %d", code)
	}
	childID := firstID(t, out)
	if code, _, _ = runCLI(t, db, "update", childID, "--assignee", "bob", "--add-label", "backend", "--set-metadata", "team=platform", "--json"); code != 0 {
		t.Fatalf("update child failed")
	}

	if code, _, _ = runCLI(t, db, "show", aID, "--json"); code != 0 {
		t.Fatalf("show failed")
	}
	if code, out, _ = runCLI(t, db, "show", aID); code != 0 || !strings.Contains(out, aID) {
		t.Fatalf("show human failed code=%d out=%q", code, out)
	}
	if code, out, _ = runCLI(t, db, "list"); code != 0 || !strings.Contains(out, "Found 3 issue(s):") || !strings.Contains(out, aID) || strings.Contains(out, "\"id\"") {
		t.Fatalf("list human output failed code=%d out=%q", code, out)
	}
	if code, out, _ = runCLI(t, db, "list", "--json", "--flat", "--no-pager"); code != 0 || !strings.Contains(out, aID) {
		t.Fatalf("list failed out=%q", out)
	}

	if code, _, _ = runCLI(t, db, "update", aID, "--status", "in_progress", "--assignee", "alice", "--add-label", "x", "--set-metadata", "k=v", "--json"); code != 0 {
		t.Fatalf("update failed")
	}
	if code, _, _ = runCLI(t, db, "label", "add", aID, "triage", "--json"); code != 0 {
		t.Fatalf("label add failed")
	}
	if code, out, _ = runCLI(t, db, "label", "add", aID, "human"); code != 0 || !strings.Contains(out, "Added label") {
		t.Fatalf("label add human failed code=%d out=%q", code, out)
	}
	if code, _, _ = runCLI(t, db, "label", "list", aID, "--json"); code != 0 {
		t.Fatalf("label list failed")
	}
	if code, out, _ = runCLI(t, db, "label", "list", aID); code != 0 || !strings.Contains(out, "Labels for") {
		t.Fatalf("label list human failed code=%d out=%q", code, out)
	}
	if code, _, _ = runCLI(t, db, "label", "remove", aID, "triage", "--json"); code != 0 {
		t.Fatalf("label remove failed")
	}
	if code, out, _ = runCLI(t, db, "label", "remove", aID, "human"); code != 0 || !strings.Contains(out, "Removed label") {
		t.Fatalf("label remove human failed code=%d out=%q", code, out)
	}
	if code, _, _ = runCLI(t, db, "label", "list-all", "--json"); code != 0 {
		t.Fatalf("label list-all failed")
	}
	if code, out, _ = runCLI(t, db, "label", "list-all"); code != 0 || !strings.Contains(out, "All labels") {
		t.Fatalf("label list-all human failed code=%d out=%q", code, out)
	}

	if code, _, _ = runCLI(t, db, "dep", "add", aID, bID, "--json"); code != 0 {
		t.Fatalf("dep add failed")
	}
	if code, _, _ = runCLI(t, db, "dep", "list", aID, "--json"); code != 0 {
		t.Fatalf("dep list failed")
	}
	if code, _, _ = runCLI(t, db, "dep", "remove", aID, bID, "--json"); code != 0 {
		t.Fatalf("dep remove failed")
	}

	if code, _, _ = runCLI(t, db, "comments", "add", aID, "hello", "--json"); code != 0 {
		t.Fatalf("comments add failed")
	}
	commentFile := filepath.Join(t.TempDir(), "comment.txt")
	if err := os.WriteFile(commentFile, []byte("from file"), 0o644); err != nil {
		t.Fatalf("write comment file: %v", err)
	}
	if code, _, _ = runCLI(t, db, "comments", "add", aID, "-f", commentFile, "--author", "alice", "--json"); code != 0 {
		t.Fatalf("comments add from file failed")
	}
	if code, _, _ = runCLI(t, db, "comments", aID, "--json"); code != 0 {
		t.Fatalf("comments list failed")
	}

	if code, out, _ = runCLI(t, db, "todo", "add", "todo item", "--json"); code != 0 {
		t.Fatalf("todo add failed")
	}
	todoID := firstID(t, out)
	if code, _, _ = runCLI(t, db, "todo", "--json"); code != 0 {
		t.Fatalf("todo list failed")
	}
	if code, _, _ = runCLI(t, db, "todo", "done", todoID, "--json"); code != 0 {
		t.Fatalf("todo done failed")
	}

	if code, out, _ = runCLI(t, db, "ready", "--assignee", "bob", "--label", "backend", "--metadata-field", "team=platform", "--parent", aID, "--type", "task", "--priority", "2", "--limit", "1"); code != 0 || !strings.Contains(out, "Found 1 ready issue(s):") || !strings.Contains(out, childID) || strings.Contains(out, "\"id\": \""+bID+"\"") {
		t.Fatalf("ready human output failed code=%d out=%q", code, out)
	}
	if code, out, _ = runCLI(t, db, "ready", "--assignee", "bob", "--label", "backend", "--metadata-field", "team=platform", "--parent", aID, "--type", "task", "--priority", "2", "--limit", "1", "--json"); code != 0 || !strings.Contains(out, childID) || strings.Contains(out, "\"id\": \""+bID+"\"") {
		t.Fatalf("ready filtered failed code=%d out=%q", code, out)
	}
	if code, out, _ = runCLI(t, db, "ready", "--unassigned", "--json"); code != 0 || !strings.Contains(out, bID) || strings.Contains(out, childID) {
		t.Fatalf("ready unassigned failed code=%d out=%q", code, out)
	}

	if code, out, _ = runCLI(t, db, "children", aID, "--json"); code != 0 || !strings.Contains(out, childID) {
		t.Fatalf("children failed out=%q", out)
	}

	if code, _, _ = runCLI(t, db, "dep", "add", bID, childID, "--json"); code != 0 {
		t.Fatalf("dep add blocker failed")
	}
	if code, out, _ = runCLI(t, db, "blocked", "--json"); code != 0 || !strings.Contains(out, childID) {
		t.Fatalf("blocked failed out=%q", out)
	}

	if code, out, _ = runCLI(t, db, "create", "dup", "--type", "bug", "--json"); code != 0 {
		t.Fatalf("create dup failed")
	}
	dupID := firstID(t, out)
	if code, _, _ = runCLI(t, db, "duplicate", dupID, "--of", aID, "--json"); code != 0 {
		t.Fatalf("duplicate failed")
	}

	if code, out, _ = runCLI(t, db, "create", "replacement", "--type", "bug", "--json"); code != 0 {
		t.Fatalf("create replacement failed")
	}
	replID := firstID(t, out)
	if code, out, _ = runCLI(t, db, "create", "Manual gate", "--type", "gate", "--json"); code != 0 {
		t.Fatalf("create gate failed")
	}
	gateID := firstID(t, out)
	if code, _, _ = runCLI(t, db, "supersede", aID, "--with", replID, "--json"); code != 0 {
		t.Fatalf("supersede failed")
	}

	if code, _, _ = runCLI(t, db, "types", "--json"); code != 0 {
		t.Fatalf("types failed")
	}
	if code, out, _ = runCLI(t, db, "query", "status=open AND priority<=2", "--json"); code != 0 || !strings.Contains(out, "\"id\"") {
		t.Fatalf("query json failed code=%d out=%q", code, out)
	}
	if code, out, _ = runCLI(t, db, "query", "status=open AND priority<=2"); code != 0 || !strings.Contains(out, "Found") {
		t.Fatalf("query human failed code=%d out=%q", code, out)
	}
	if code, _, _ = runCLI(t, db, "search", "Task", "--json", "-n", "3"); code != 0 {
		t.Fatalf("search failed")
	}
	if code, out, _ = runCLI(t, db, "count"); code != 0 || strings.Contains(out, `"count"`) {
		t.Fatalf("count human failed code=%d out=%q", code, out)
	}
	if code, out, _ = runCLI(t, db, "count", "--status", "open", "--json"); code != 0 || !strings.Contains(out, `"count"`) {
		t.Fatalf("count failed code=%d out=%q", code, out)
	}
	if code, out, _ = runCLI(t, db, "gate", "list"); code != 0 || !strings.Contains(out, "Found") {
		t.Fatalf("gate list human failed code=%d out=%q", code, out)
	}
	if code, out, _ = runCLI(t, db, "gate", "list", "--json"); code != 0 || !strings.Contains(out, gateID) {
		t.Fatalf("gate list json failed code=%d out=%q", code, out)
	}
	if code, out, _ = runCLI(t, db, "gate", "show", gateID); code != 0 || !strings.Contains(out, "Gate "+gateID) {
		t.Fatalf("gate show human failed code=%d out=%q", code, out)
	}
	if code, out, _ = runCLI(t, db, "gate", "check", "--json"); code != 0 || !strings.Contains(out, "open_gates") {
		t.Fatalf("gate check json failed code=%d out=%q", code, out)
	}
	if code, out, _ = runCLI(t, db, "gate", "check"); code != 0 || !strings.Contains(out, "Gate check complete") {
		t.Fatalf("gate check human failed code=%d out=%q", code, out)
	}
	if code, out, _ = runCLI(t, db, "gate", "resolve", gateID, "--reason", "manual"); code != 0 || !strings.Contains(out, "Resolved gate") {
		t.Fatalf("gate resolve human failed code=%d out=%q", code, out)
	}
	if code, out, _ = runCLI(t, db, "gate", "list", "--all", "--json"); code != 0 || !strings.Contains(out, gateID) {
		t.Fatalf("gate list --all json failed code=%d out=%q", code, out)
	}
	if code, out, _ = runCLI(t, db, "backup", "--json"); code != 0 || !strings.Contains(out, "backup_path") {
		t.Fatalf("backup export failed code=%d out=%q", code, out)
	}
	var backupPayload map[string]any
	if err := json.Unmarshal([]byte(out), &backupPayload); err != nil {
		t.Fatalf("backup json decode failed: %v", err)
	}
	backupPath, _ := backupPayload["backup_path"].(string)
	if backupPath == "" {
		t.Fatalf("missing backup_path in payload=%q", out)
	}
	if code, out, _ = runCLI(t, db, "backup", "status", "--json"); code != 0 || !strings.Contains(out, "latest_backup") {
		t.Fatalf("backup status failed code=%d out=%q", code, out)
	}
	if code, out, _ = runCLI(t, db, "backup", "restore", backupPath, "--json"); code != 0 || !strings.Contains(out, "restored_from") {
		t.Fatalf("backup restore failed code=%d out=%q", code, out)
	}
	if code, out, _ = runCLI(t, db, "status", "--json"); code != 0 || !strings.Contains(out, "\"summary\"") || !strings.Contains(out, "\"open\"") || !strings.Contains(out, "\"closed\"") {
		t.Fatalf("status failed code=%d out=%q", code, out)
	}
	if code, out, _ = runCLI(t, db, "status"); code != 0 || !strings.Contains(out, "Issue Database Status") || !strings.Contains(out, "Ready to Work") {
		t.Fatalf("status human failed code=%d out=%q", code, out)
	}
	if code, out, _ = runCLI(t, db, "stats", "--json"); code != 0 || !strings.Contains(out, "\"summary\"") {
		t.Fatalf("stats alias failed code=%d out=%q", code, out)
	}

	if code, _, _ = runCLI(t, db, "reopen", aID, "--json"); code != 0 {
		t.Fatalf("reopen failed")
	}
	if code, _, _ = runCLI(t, db, "close", bID, "--reason", "done", "--json"); code != 0 {
		t.Fatalf("close failed")
	}
	if code, _, _ = runCLI(t, db, "delete", bID, "--force", "--json"); code != 0 {
		t.Fatalf("delete failed")
	}
}

func TestRun_ErrorFlagsAndValidation(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	_, _, _ = runCLI(t, db, "init", "--json")
	_, _, _ = runCLI(t, db, "create", "seed", "--json")

	cases := [][]string{
		{"create", "x", "--priority", "bad", "--json"},
		{"create", "x", "--deps", ":", "--json"},
		{"list", "--bogus"},
		{"show", "--bogus"},
		{"update"},
		{"label"},
		{"dep"},
		{"comments"},
		{"todo", "nope"},
		{"children"},
		{"blocked", "--bad"},
		{"duplicate", "x", "--json"},
		{"supersede", "x", "--json"},
		{"types", "--bad"},
		{"gate", "wat"},
		{"gate", "show"},
		{"gate", "resolve"},
		{"backup", "wat"},
	}
	for _, c := range cases {
		code, _, _ := runCLI(t, db, c...)
		if code == 0 {
			t.Fatalf("expected failure for %v", c)
		}
	}

	code, _, _ := runCLI(t, db, "query", "madeupfield=1", "--json")
	if code == 0 {
		t.Fatalf("expected bad query to fail")
	}
}

func TestRun_BackupCommandBranches(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	if code, _, _ := runCLI(t, db, "init", "--json"); code != 0 {
		t.Fatalf("init failed")
	}
	if code, out, _ := runCLI(t, db, "backup", "status"); code != 0 || !strings.Contains(out, "No backups found") {
		t.Fatalf("backup status human failed code=%d out=%q", code, out)
	}
	code, out, _ := runCLI(t, db, "backup", "--json")
	if code != 0 || !strings.Contains(out, "backup_path") {
		t.Fatalf("backup default export failed code=%d out=%q", code, out)
	}
	var backupPayload map[string]any
	if err := json.Unmarshal([]byte(out), &backupPayload); err != nil {
		t.Fatalf("decode backup payload failed: %v", err)
	}
	backupPath, _ := backupPayload["backup_path"].(string)
	if backupPath == "" {
		t.Fatalf("missing backup_path in payload: %q", out)
	}
	if code, out, _ := runCLI(t, db, "backup"); code != 0 || !strings.Contains(out, "Backup created") {
		t.Fatalf("backup human export failed code=%d out=%q", code, out)
	}
	if code, out, _ := runCLI(t, db, "backup", "status", "--json"); code != 0 || !strings.Contains(out, "latest_backup") {
		t.Fatalf("backup status json failed code=%d out=%q", code, out)
	}
	if code, out, _ := runCLI(t, db, "backup", "status"); code != 0 || !strings.Contains(out, "Latest backup") {
		t.Fatalf("backup status human failed code=%d out=%q", code, out)
	}
	if code, out, _ := runCLI(t, db, "backup", "restore", backupPath, "--json"); code != 0 || !strings.Contains(out, "restored_from") {
		t.Fatalf("backup restore explicit failed code=%d out=%q", code, out)
	}
	if code, out, _ := runCLI(t, db, "backup", "restore", "--json"); code != 0 || !strings.Contains(out, "restored_from") {
		t.Fatalf("backup restore latest failed code=%d out=%q", code, out)
	}
	if code, _, errOut := runCLI(t, db, "backup", "restore", "/definitely/missing.sqlite"); code == 0 || !strings.Contains(strings.ToLower(errOut), "failed") {
		t.Fatalf("expected restore bad path error, code=%d err=%q", code, errOut)
	}
	if code, _, errOut := runCLI(t, db, "backup", "init", "/tmp/foo"); code == 0 || !strings.Contains(errOut, "unsupported") {
		t.Fatalf("expected backup init unsupported code=%d err=%q", code, errOut)
	}
	if code, _, errOut := runCLI(t, db, "backup", "sync"); code == 0 || !strings.Contains(errOut, "unsupported") {
		t.Fatalf("expected backup sync unsupported code=%d err=%q", code, errOut)
	}
	if code, out, _ := runCLI(t, db, "backup", "--help"); code != 0 || !strings.Contains(out, "sq backup") {
		t.Fatalf("backup --help failed code=%d out=%q", code, out)
	}
}

func TestRun_GateCommandAdditionalBranches(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	if code, _, _ := runCLI(t, db, "init", "--json"); code != 0 {
		t.Fatalf("init failed")
	}
	code, out, _ := runCLI(t, db, "create", "gate branch", "--type", "gate", "--json")
	if code != 0 {
		t.Fatalf("create gate failed: %d", code)
	}
	gateID := firstID(t, out)

	if code, out, _ := runCLI(t, db, "gate", "show", gateID, "--json"); code != 0 || !strings.Contains(out, gateID) {
		t.Fatalf("gate show json failed code=%d out=%q", code, out)
	}
	if code, out, _ := runCLI(t, db, "gate", "resolve", gateID, "--json"); code != 0 || !strings.Contains(out, "\"status\": \"closed\"") {
		t.Fatalf("gate resolve json failed code=%d out=%q", code, out)
	}
	if code, out, _ := runCLI(t, db, "gate", "list", "--all"); code != 0 || !strings.Contains(out, gateID) {
		t.Fatalf("gate list --all human failed code=%d out=%q", code, out)
	}
	if code, _, errOut := runCLI(t, db, "gate", "show"); code == 0 || !strings.Contains(strings.ToLower(errOut), "usage") {
		t.Fatalf("expected gate show usage failure code=%d err=%q", code, errOut)
	}
}

func TestRun_BackupCommandErrorBranches(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	if code, _, _ := runCLI(t, db, "init", "--json"); code != 0 {
		t.Fatalf("init failed")
	}
	if code, _, errOut := runCLI(t, db, "backup", "wat"); code == 0 || !strings.Contains(strings.ToLower(errOut), "unknown backup subcommand") {
		t.Fatalf("expected unknown backup subcommand failure code=%d err=%q", code, errOut)
	}
	if code, _, errOut := runCLI(t, db, "backup", "restore", "/definitely/missing.sqlite", "--json"); code == 0 || !strings.Contains(strings.ToLower(errOut), "failed") {
		t.Fatalf("expected missing backup restore failure code=%d err=%q", code, errOut)
	}
}

func TestRun_GateShowRejectsNonGate(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	if code, _, _ := runCLI(t, db, "init", "--json"); code != 0 {
		t.Fatalf("init failed")
	}
	code, out, _ := runCLI(t, db, "create", "not a gate", "--type", "task", "--json")
	if code != 0 {
		t.Fatalf("create failed: %d", code)
	}
	id := firstID(t, out)
	code, _, errOut := runCLI(t, db, "gate", "show", id)
	if code == 0 || !strings.Contains(errOut, "not a gate") {
		t.Fatalf("expected non-gate rejection, code=%d err=%q", code, errOut)
	}
}

func TestRun_JSONOutputParseable(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	if code, _, _ := runCLI(t, db, "init", "--json"); code != 0 {
		t.Fatalf("init failed")
	}
	code, out, _ := runCLI(t, db, "types", "--json")
	if code != 0 {
		t.Fatalf("types failed")
	}
	dec := json.NewDecoder(bytes.NewBufferString(out))
	var payload map[string]any
	if err := dec.Decode(&payload); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if _, ok := payload["core_types"]; !ok {
		t.Fatalf("missing core_types")
	}
}

func TestRun_QVariants(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	if code, _, _ := runCLI(t, db, "init", "--json"); code != 0 {
		t.Fatalf("init failed")
	}

	code, out, _ := runCLI(t, db, "q", "Quick One", "--type", "bug", "--priority", "3", "--description", "d", "--json")
	if code != 0 {
		t.Fatalf("q --json failed: %d", code)
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(out), &obj); err != nil {
		t.Fatalf("q --json not object json: %v (%q)", err, out)
	}
	if _, ok := obj["id"]; !ok {
		t.Fatalf("q --json missing id: %q", out)
	}

	code, out, _ = runCLI(t, db, "q", "Quick Two")
	if code != 0 {
		t.Fatalf("q plain failed: %d", code)
	}
	if strings.TrimSpace(out) == "" {
		t.Fatalf("q plain expected id output")
	}

	if code, _, errOut := runCLI(t, db, "q", "Bad", "--priority", "NaN"); code != 2 || !strings.Contains(errOut, "invalid --priority") {
		t.Fatalf("expected invalid priority usage failure, code=%d err=%q", code, errOut)
	}
	if code, _, errOut := runCLI(t, db, "q", "Bad", "--unknown"); code != 2 || !strings.Contains(errOut, "unknown flag") {
		t.Fatalf("expected unknown flag usage failure, code=%d err=%q", code, errOut)
	}
}
