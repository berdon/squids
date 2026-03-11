package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCap(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr
	code := run(args)
	_ = wOut.Close()
	_ = wErr.Close()
	outB, _ := io.ReadAll(rOut)
	errB, _ := io.ReadAll(rErr)
	os.Stdout, os.Stderr = oldOut, oldErr
	return code, string(outB), string(errB)
}

func idFromJSON(t *testing.T, out string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("bad json: %v out=%q", err, out)
	}
	id, _ := m["id"].(string)
	if id == "" {
		t.Fatalf("missing id out=%q", out)
	}
	return id
}

func TestRun_IntegrationFlow(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	old := os.Getenv("SQ_DB_PATH")
	_ = os.Setenv("SQ_DB_PATH", db)
	defer func() { _ = os.Setenv("SQ_DB_PATH", old) }()

	ok := func(args ...string) string {
		t.Helper()
		code, out, err := runCap(t, args...)
		if code != 0 {
			t.Fatalf("expected ok for %v got %d err=%q", args, code, err)
		}
		return out
	}
	bad := func(args ...string) {
		t.Helper()
		if code, _, _ := runCap(t, args...); code == 0 {
			t.Fatalf("expected failure for %v", args)
		}
	}

	ok("init", "--json")
	ok("ready", "--json")
	aID := idFromJSON(t, ok("create", "A", "--json"))
	bID := idFromJSON(t, ok("create", "B", "--json"))

	ok("show", aID, "--json")
	ok("list", "--json", "--flat", "--no-pager")
	ok("update", aID, "--status", "in_progress", "--assignee", "alice", "--json")
	ok("label", "add", aID, "x", "--json")
	ok("dep", "add", aID, bID, "--json")
	ok("comments", "add", aID, "hi", "--json")
	ok("comments", aID, "--json")
	ok("query", "status=in_progress", "--json")
	ok("search", "A", "--json")
	ok("count", "--json")
	ok("status", "--json")
	ok("types", "--json")

	todoID := idFromJSON(t, ok("todo", "add", "x", "--json"))
	ok("todo", "done", todoID, "--json")
	ok("children", aID, "--json")
	ok("blocked", "--json")

	dupID := idFromJSON(t, ok("create", "dup", "--json"))
	ok("duplicate", dupID, "--of", aID, "--json")
	replID := idFromJSON(t, ok("create", "repl", "--json"))
	ok("supersede", aID, "--with", replID, "--json")

	ok("reopen", aID, "--json")
	ok("close", aID, "--reason", "done", "--json")
	ok("delete", bID, "--force", "--json")

	bad("unknown-command")
	if _, _, err := runCap(t, "help"); !strings.Contains(err, "") {
		// no-op; just execute help path in this package test binary
	}
}
