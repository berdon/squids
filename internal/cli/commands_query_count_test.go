package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/berdon/squids/internal/store"
)

func TestCmdCountHelpAndUnknownFlag(t *testing.T) {
	code, out, _ := captureOutput(t, func() int { return cmdCount([]string{"--help"}) })
	if code != 0 {
		t.Fatalf("expected help exit 0, got %d", code)
	}
	if !strings.Contains(out, "Count issues matching the specified filters.") || !strings.Contains(out, "Global Flags:") {
		t.Fatalf("unexpected help output: %q", out)
	}

	code, _, errOut := captureOutput(t, func() int { return cmdCount([]string{"--wat"}) })
	if code != 2 {
		t.Fatalf("expected usage exit 2, got %d", code)
	}
	if !strings.Contains(errOut, "unknown flag") {
		t.Fatalf("expected unknown flag error, got %q", errOut)
	}
}

func TestCmdCountSupportsGlobalCompatFlags(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "tasks.sqlite")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := store.Init(db); err != nil {
		t.Fatalf("init db: %v", err)
	}
	if _, err := store.CreateTask(db, store.CreateInput{Title: "count me", IssueType: "task", Priority: 1}); err != nil {
		t.Fatalf("create task: %v", err)
	}

	oldDB := os.Getenv("SQ_DB_PATH")
	_ = os.Setenv("SQ_DB_PATH", dbPath)
	defer func() { _ = os.Setenv("SQ_DB_PATH", oldDB) }()

	code, out, errOut := captureOutput(t, func() int {
		return cmdCount([]string{"--db", dbPath, "--actor", "tester", "--dolt-auto-commit", "off", "--quiet", "--status", "open", "--json"})
	})
	if code != 0 {
		t.Fatalf("expected success, got code=%d stderr=%q", code, errOut)
	}
	if !strings.Contains(out, `"count": 1`) {
		t.Fatalf("expected json count output, got %q", out)
	}

	code, out, errOut = captureOutput(t, func() int {
		return cmdCount([]string{"--verbose", "--profile", "--readonly", "--sandbox", "--status", "open"})
	})
	if code != 0 {
		t.Fatalf("expected success for plain count, got code=%d stderr=%q", code, errOut)
	}
	if strings.TrimSpace(out) != "1" {
		t.Fatalf("expected plain count output, got %q", out)
	}
}
