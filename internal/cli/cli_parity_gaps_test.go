package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestInitHelpAndValidation(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")

	code, out, errOut := runCLI(t, db, "init", "--help")
	if code != 0 || !strings.Contains(out, "Usage:") || !strings.Contains(out, "sq init [flags]") {
		t.Fatalf("init --help failed code=%d out=%q err=%q", code, out, errOut)
	}

	code, _, errOut = runCLI(t, db, "init", "extra")
	if code != 2 || !strings.Contains(errOut, "does not accept positional arguments") {
		t.Fatalf("init positional validation failed code=%d err=%q", code, errOut)
	}

	code, _, errOut = runCLI(t, db, "init", "--db")
	if code != 2 || !strings.Contains(errOut, "missing value for --db") {
		t.Fatalf("init missing db value failed code=%d err=%q", code, errOut)
	}

	code, out, errOut = runCLI(t, db, "init", "--prefix", "bd", "--json")
	if code != 0 || !strings.Contains(out, `"command": "init"`) {
		t.Fatalf("init with prefix compatibility failed code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestDepHelpAndValidation(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	if code, _, _ := runCLI(t, db, "init", "--json"); code != 0 {
		t.Fatalf("init failed")
	}
	parentID := firstID(t, mustRunCLI(t, db, "create", "A", "--json"))
	childID := firstID(t, mustRunCLI(t, db, "create", "B", "--json"))

	code, out, errOut := runCLI(t, db, "dep", "add", "--help")
	if code != 0 || !strings.Contains(out, "sq dep add <issue-id> <depends-on-id> [flags]") {
		t.Fatalf("dep add --help failed code=%d out=%q err=%q", code, out, errOut)
	}

	code, out, errOut = runCLI(t, db, "dep", "remove", "--help")
	if code != 0 || !strings.Contains(out, "sq dep remove <issue-id> <depends-on-id> [flags]") {
		t.Fatalf("dep remove --help failed code=%d out=%q err=%q", code, out, errOut)
	}

	code, out, errOut = runCLI(t, db, "dep", "list", "--help")
	if code != 0 || !strings.Contains(out, "sq dep list <issue-id> [flags]") {
		t.Fatalf("dep list --help failed code=%d out=%q err=%q", code, out, errOut)
	}

	code, _, errOut = runCLI(t, db, "dep", "add", parentID, childID, "--wat")
	if code != 2 || !strings.Contains(errOut, "unknown flag: --wat") {
		t.Fatalf("dep add unknown flag failed code=%d err=%q", code, errOut)
	}

	if code, _, errOut = runCLI(t, db, "dep", "add", parentID, childID, "--json"); code != 0 {
		t.Fatalf("dep add failed code=%d err=%q", code, errOut)
	}
	code, _, errOut = runCLI(t, db, "dep", "remove", parentID, childID, "extra")
	if code != 2 || !strings.Contains(errOut, "usage: sq dep remove <issue-id> <depends-on-id> [--json]") {
		t.Fatalf("dep remove extra arg failed code=%d err=%q", code, errOut)
	}
	code, _, errOut = runCLI(t, db, "dep", "list", parentID, "extra")
	if code != 2 || !strings.Contains(errOut, "usage: sq dep list <issue-id> [--json]") {
		t.Fatalf("dep list extra arg failed code=%d err=%q", code, errOut)
	}
}

func TestTypesAndBlockedHumanPaths(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	if code, _, _ := runCLI(t, db, "init", "--json"); code != 0 {
		t.Fatalf("init failed")
	}

	code, out, errOut := runCLI(t, db, "types", "--help")
	if code != 0 || !strings.Contains(out, "sq types [flags]") || !strings.Contains(out, "Global Flags:") {
		t.Fatalf("types --help failed code=%d out=%q err=%q", code, out, errOut)
	}
	code, out, errOut = runCLI(t, db, "types")
	if code != 0 || !strings.Contains(out, "Core issue types:") || !strings.Contains(out, "epic") {
		t.Fatalf("types human output failed code=%d out=%q err=%q", code, out, errOut)
	}
	code, _, errOut = runCLI(t, db, "types", "extra")
	if code != 2 || !strings.Contains(errOut, "types does not accept positional arguments") {
		t.Fatalf("types positional validation failed code=%d err=%q", code, errOut)
	}

	code, out, errOut = runCLI(t, db, "blocked", "--help")
	if code != 0 || !strings.Contains(out, "sq blocked [flags]") {
		t.Fatalf("blocked --help failed code=%d out=%q err=%q", code, out, errOut)
	}
	code, out, errOut = runCLI(t, db, "blocked")
	if code != 0 || !strings.Contains(out, "No blocked issues found") {
		t.Fatalf("blocked human output failed code=%d out=%q err=%q", code, out, errOut)
	}
	code, _, errOut = runCLI(t, db, "blocked", "--parent")
	if code != 2 || !strings.Contains(errOut, "missing value for --parent") {
		t.Fatalf("blocked parent validation failed code=%d err=%q", code, errOut)
	}
}

func TestCompatHelpAndSurfaceBranches(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	if code, _, _ := runCLI(t, db, "init", "--json"); code != 0 {
		t.Fatalf("init failed")
	}

	gateID := firstID(t, mustRunCLI(t, db, "create", "Gate", "--type", "gate", "--json"))

	code, out, errOut := runCLI(t, db, "gate", "list", "--help")
	if code != 0 || !strings.Contains(out, "sq gate list [flags]") {
		t.Fatalf("gate list help failed code=%d out=%q err=%q", code, out, errOut)
	}
	code, out, errOut = runCLI(t, db, "gate", "show", gateID, "--help")
	if code != 0 || !strings.Contains(out, "sq gate show <id> [flags]") {
		t.Fatalf("gate show help failed code=%d out=%q err=%q", code, out, errOut)
	}

	code, out, errOut = runCLI(t, db, "import-beads", "--help")
	if code != 0 || !strings.Contains(out, "sq import-beads [flags]") {
		t.Fatalf("import-beads help failed code=%d out=%q err=%q", code, out, errOut)
	}

	code, out, errOut = runCLI(t, db, "gitlab")
	if code != 0 || !strings.Contains(out, "sq gitlab [projects|status|sync]") {
		t.Fatalf("gitlab root help failed code=%d out=%q err=%q", code, out, errOut)
	}
	for _, cmd := range [][]string{{"gitlab", "projects", "--json"}, {"mol", "--json"}, {"mail", "--json"}, {"setup", "--json"}, {"purge", "--json"}} {
		code, _, _ = runCLI(t, db, cmd...)
		if code == 0 {
			t.Fatalf("expected non-zero for compat surface %v", cmd)
		}
	}
}
