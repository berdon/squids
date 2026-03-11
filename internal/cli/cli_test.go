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
	if code != 0 || !strings.Contains(out, "sq - squids task CLI") {
		t.Fatalf("help failed code=%d out=%q", code, out)
	}
	code, _, _ = runCLI(t, db, "-h")
	if code != 0 {
		t.Fatalf("-h failed code=%d", code)
	}
	code, _, _ = runCLI(t, db, "--help")
	if code != 0 {
		t.Fatalf("--help failed code=%d", code)
	}

	code, _, err := runCLI(t, db, "nope")
	if code != 2 || !strings.Contains(err, "unknown command") {
		t.Fatalf("unknown command failed code=%d err=%q", code, err)
	}
}

func TestRun_EndToEndCommandFamilies(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")

	if code, _, _ := runCLI(t, db, "init", "--json"); code != 0 {
		t.Fatalf("init failed: %d", code)
	}
	if code, _, _ := runCLI(t, db, "ready", "--json"); code != 0 {
		t.Fatalf("ready failed: %d", code)
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

	if code, _, _ = runCLI(t, db, "show", aID, "--json"); code != 0 {
		t.Fatalf("show failed")
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
	if code, _, _ = runCLI(t, db, "label", "list", aID, "--json"); code != 0 {
		t.Fatalf("label list failed")
	}
	if code, _, _ = runCLI(t, db, "label", "remove", aID, "triage", "--json"); code != 0 {
		t.Fatalf("label remove failed")
	}
	if code, _, _ = runCLI(t, db, "label", "list-all", "--json"); code != 0 {
		t.Fatalf("label list-all failed")
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
	if code, _, _ = runCLI(t, db, "supersede", aID, "--with", replID, "--json"); code != 0 {
		t.Fatalf("supersede failed")
	}

	if code, _, _ = runCLI(t, db, "types", "--json"); code != 0 {
		t.Fatalf("types failed")
	}
	if code, _, _ = runCLI(t, db, "query", "status=open AND priority<=2", "--json"); code != 0 {
		t.Fatalf("query failed")
	}
	if code, _, _ = runCLI(t, db, "search", "Task", "--json", "-n", "3"); code != 0 {
		t.Fatalf("search failed")
	}
	if code, _, _ = runCLI(t, db, "count", "--status", "open", "--json"); code != 0 {
		t.Fatalf("count failed")
	}
	if code, _, _ = runCLI(t, db, "status", "--json"); code != 0 {
		t.Fatalf("status failed")
	}
	if code, _, _ = runCLI(t, db, "stats", "--json"); code != 0 {
		t.Fatalf("stats alias failed")
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
