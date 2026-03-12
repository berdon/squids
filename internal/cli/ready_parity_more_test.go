package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestReadyParityFlagsAndSorting(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	if code, _, err := runCLI(t, db, "init", "--json"); code != 0 {
		t.Fatalf("init failed code=%d err=%q", code, err)
	}

	code, out, _ := runCLI(t, db, "create", "First", "--priority", "2", "--json")
	if code != 0 {
		t.Fatalf("create first failed: %d", code)
	}
	firstTaskID := firstID(t, out)
	code, out, _ = runCLI(t, db, "create", "Second", "--priority", "1", "--json")
	if code != 0 {
		t.Fatalf("create second failed: %d", code)
	}
	secondID := firstID(t, out)
	if code, _, _ = runCLI(t, db, "update", secondID, "--add-label", "cli", "--set-metadata", "team=platform", "--json"); code != 0 {
		t.Fatalf("update second failed")
	}

	code, out, errOut := runCLI(t, db, "ready", "--label-any", "cli,other", "--has-metadata-key", "team", "--sort", "oldest", "--gated", "--include-deferred", "--include-ephemeral", "--plain", "--pretty", "--mol", "x", "--mol-type", "work", "--rig", "bd", "--json")
	if code != 0 || errOut != "" || !strings.Contains(out, secondID) || strings.Contains(out, firstTaskID) {
		t.Fatalf("ready flags failed code=%d out=%q err=%q", code, out, errOut)
	}

	code, out, errOut = runCLI(t, db, "ready", "--sort", "oldest", "--json")
	if code != 0 || errOut != "" || strings.Index(out, firstTaskID) > strings.Index(out, secondID) {
		t.Fatalf("ready oldest sort failed code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestReadyParityUsageFailures(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	cases := [][]string{
		{"ready", "thing"},
		{"ready", "--assignee"},
		{"ready", "--limit", "x"},
		{"ready", "--priority", "x"},
		{"ready", "--sort"},
	}
	for _, args := range cases {
		code, _, errOut := runCLI(t, db, args...)
		if code != 2 || errOut == "" {
			t.Fatalf("expected usage failure for %v code=%d err=%q", args, code, errOut)
		}
	}
}
