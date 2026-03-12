package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI_CommandBranchCoverage(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")

	mustOK := func(args ...string) string {
		t.Helper()
		code, out, err := runCLI(t, db, args...)
		if code != 0 {
			t.Fatalf("expected ok for %v code=%d err=%q", args, code, err)
		}
		return out
	}
	mustFail := func(args ...string) {
		t.Helper()
		code, _, _ := runCLI(t, db, args...)
		if code == 0 {
			t.Fatalf("expected failure for %v", args)
		}
	}

	mustOK("help")
	mustOK("help", "create")
	mustOK("help", "--all")
	mustOK("help", "--help")
	mustOK("help", "--actor", "tester")
	mustFail("help", "--wat")
	mustFail("help", "create", "list")
	mustOK("init", "--json")
	mustOK("ready", "--json")

	id := firstID(t, mustOK("create", "A", "--json"))
	id2 := firstID(t, mustOK("create", "B", "--json"))

	mustFail("q")
	mustFail("q", "x", "--priority", "bad")
	mustFail("q", "x", "--wat")
	qraw := mustOK("q", "quickone")
	if !strings.Contains(qraw, "bd-") {
		t.Fatalf("expected q output id, got %q", qraw)
	}
	mustOK("q", "quickjson", "--json")

	mustFail("create")
	mustFail("create", "x", "--priority", "nope", "--json")
	mustFail("create", "x", "--wat")

	mustFail("show")
	mustFail("show", "missing", "--wat")

	mustFail("list", "--wat")

	mustFail("update")
	mustFail("update", id, "--set-metadata", "bad", "--json")
	mustFail("update", id, "--wat")
	mustOK("update", id, "--json")

	mustFail("close")
	mustFail("close", id, "--wat")
	mustOK("close", id, "--json")

	mustFail("reopen")
	mustFail("reopen", id, "--wat")
	mustOK("reopen", id, "--json")

	mustFail("delete")
	mustFail("delete", id2, "--wat")

	mustFail("label")
	mustFail("label", "add", id)
	mustFail("label", "list")
	mustFail("label", "remove", id)
	mustFail("label", "wat")
	mustOK("label", "add", id, "x", "--json")
	mustOK("label", "list", id, "--json")
	mustOK("label", "remove", id, "x", "--json")
	mustOK("label", "list-all", "--json")

	mustFail("dep")
	mustFail("dep", "add", id)
	mustFail("dep", "list")
	mustFail("dep", "remove", id)
	mustFail("dep", "wat")
	mustOK("dep", "add", id, id2, "--json")
	mustOK("dep", "list", id, "--json")
	mustOK("dep", "rm", id, id2, "--json")

	mustFail("comments")
	mustFail("comments", "add", id)
	mustOK("comments", "add", id, "hi", "--json")
	mustOK("comments", id, "--json")

	mustFail("todo", "add")
	mustFail("todo", "add", "x", "--priority", "bad", "--json")
	mustFail("todo", "add", "x", "--wat")
	mustFail("todo", "done")
	mustFail("todo", "done", id, "--wat")
	mustFail("todo", "wat")
	todoID := firstID(t, mustOK("todo", "add", "x", "--description", "d", "--json"))
	mustOK("todo", "list", "--json")
	mustOK("todo", "done", todoID, "--reason", "done", "--json")

	mustFail("children")
	mustOK("children", id, "--json")

	mustFail("blocked", "--wat")
	mustOK("blocked", "--parent", id, "--json")

	mustFail("defer")
	mustFail("defer", "--json")
	mustFail("defer", id, "--wat")
	mustOK("defer", id, "--json")
	mustFail("undefer")
	mustFail("undefer", "--json")
	mustFail("undefer", id, "--wat")
	mustOK("undefer", id, "--json")

	mustFail("rename")
	mustFail("rename", id)
	mustFail("rename", id, "new", "--wat")
	mustOK("rename", id, "bd-renamed", "--json")
	id = "bd-renamed"
	mustFail("rename-prefix")
	mustFail("rename-prefix", "bd", "sq", "--wat")
	mustOK("rename-prefix", "sq", "--json")
	id = "sq-renamed"

	mustFail("duplicate")
	mustFail("duplicate", id, "--wat")
	mustFail("duplicate", id, "--json")
	dupID := firstID(t, mustOK("create", "dup", "--json"))
	mustOK("duplicate", dupID, "--of", id, "--json")

	mustFail("supersede")
	mustFail("supersede", id, "--wat")
	mustFail("supersede", id, "--json")
	replID := firstID(t, mustOK("create", "repl", "--json"))
	mustOK("supersede", id, "--with", replID, "--json")

	mustFail("types", "--wat")
	mustOK("types", "--json")

	mustFail("query")
	mustFail("query", "priority^1", "--json")
	mustOK("query", "status=open", "--json")

	mustFail("stale", "--wat")
	mustOK("stale", "--days", "1", "--json")
	mustOK("stale", "-d", "1", "--json")
	mustFail("orphans", "--wat")
	mustOK("orphans", "--json")

	mustOK("search", "x", "--query", "x", "--limit", "3", "--json")
	mustOK("search", "x", "--json", "--status", "open", "--sort", "id", "--reverse", "--long")
	mustOK("search", "x", "-x", "--json")

	mustOK("count", "--json")
	mustOK("count", "-s", "open", "--json")
	mustOK("status", "--json")
	mustFail("version", "--wat")
	v := mustOK("version")
	if !strings.Contains(v, "sq version") {
		t.Fatalf("expected plain version output, got %q", v)
	}
	mustOK("version", "--json")
	mustOK("version", "--help")
	mustOK("version", "-h")
	mustOK("version", "--quiet")
	mustOK("version", "--verbose")
	mustOK("version", "--profile")
	mustOK("version", "--readonly")
	mustOK("version", "--sandbox")
	mustOK("version", "--actor", "tester")
	mustOK("version", "--db", "/tmp/sq.db")
	mustOK("version", "--dolt-auto-commit", "off")
	mustOK("-V")
	mustOK("--version")

	mustFail("where", "--wat")
	mustOK("where")
	mustOK("where", "--json")
	mustOK("where", "--help")
	mustOK("where", "--actor", "tester")

	mustFail("info", "--wat")
	mustOK("info")
	mustOK("info", "--json")
	mustOK("info", "--schema", "--json")
	mustOK("info", "--whats-new")
	mustOK("info", "--whats-new", "--json")
	mustOK("info", "--thanks")
	mustOK("info", "--help")

	humanID := firstID(t, mustOK("create", "Human task", "--json"))
	mustOK("label", "add", humanID, "human", "--json")
	mustOK("human")
	mustOK("human", "list", "--json")
	mustOK("human", "list", "--status", "open", "--json")
	mustOK("human", "stats")
	mustFail("human", "respond", humanID)
	mustFail("human", "respond", humanID, "--wat")
	mustOK("human", "respond", humanID, "--response", "done", "--json")
	human2 := firstID(t, mustOK("create", "Human task 2", "--json"))
	mustOK("label", "add", human2, "human", "--json")
	mustFail("human", "dismiss", human2, "--wat")
	mustOK("human", "dismiss", human2, "--reason", "n/a", "--json")
	mustFail("human", "wat")

	mustOK("quickstart")
	mustOK("quickstart", "--help")
	mustOK("quickstart", "--actor", "tester")
	mustOK("quickstart", "--json")
	mustFail("quickstart", "--wat")

	mustFail("setup")
	mustFail("setup", "cursor")
	mustFail("setup", "--list")
	mustFail("setup", "cursor", "--check")
	mustFail("setup", "--wat")

	mustFail("history")
	mustFail("history", id, "--wat")
	mustFail("history", id, "--help")
	mustFail("history", id, "--json")
	mustFail("history", id, "--limit", "5", "--json")
	mustFail("history", id, "--actor", "tester", "--db", "/tmp/sq.db", "--dolt-auto-commit", "off", "--json")

	mustFail("restore")
	mustFail("restore", id, "--wat")
	mustFail("restore", id, "--help")
	mustFail("restore", id, "--json")
	mustFail("restore", id, "--actor", "tester", "--db", "/tmp/sq.db", "--dolt-auto-commit", "off", "--json")

	mustOK("audit")
	mustFail("audit", "record")
	mustFail("audit", "label")
	mustFail("audit", "record", "--wat")
	mustFail("audit", "wat")

	mustFail("set-state")
	mustFail("set-state", id)
	mustFail("set-state", id, "mode=normal")
	mustFail("set-state", id, "mode=normal", "--reason", "test", "--json")
	mustFail("set-state", id, "mode=normal", "--wat")

	mustOK("swarm")
	mustOK("swarm", "list")
	mustOK("swarm", "list", "--json")
	mustOK("swarm", "status")
	mustOK("swarm", "status", "--json")
	mustOK("swarm", "validate")
	mustOK("swarm", "validate", "--json")
	mustFail("swarm", "create")
	mustFail("swarm", "list", "--wat")
	mustFail("swarm", "wat")

	mustOK("hooks")
	mustOK("hooks", "list")
	mustOK("hooks", "list", "--json")
	mustOK("hooks", "list", "--shared", "--json")
	mustOK("hooks", "list", "--beads", "--json")
	mustOK("hooks", "install", "--json")
	mustOK("hooks", "install", "--shared", "--force", "--json")
	mustOK("hooks", "install", "--beads", "--json")
	mustOK("hooks", "uninstall")
	mustOK("hooks", "uninstall", "--json")
	mustFail("hooks", "run")
	mustOK("hooks", "run", "pre-commit")
	mustOK("hooks", "run", "pre-commit", "--json")
	mustFail("hooks", "run", "unknown")
	mustFail("hooks", "run", "wat")
	mustFail("hooks", "install", "--wat")
	mustFail("hooks", "--wat")
	mustFail("hooks", "wat")

	mustOK("onboard")
	mustOK("onboard", "--help")
	mustOK("onboard", "--actor", "tester", "--db", "/tmp/sq.db", "--dolt-auto-commit", "off")
	mustFail("onboard", "--wat")

	mustOK("completion")
	mustOK("completion", "bash")
	mustOK("completion", "zsh")
	mustOK("completion", "fish")
	mustOK("completion", "powershell")
	mustOK("completion", "--help")
	mustOK("completion", "bash", "--json")
	mustOK("completion", "bash", "--actor", "tester", "--db", "/tmp/sq.db", "--dolt-auto-commit", "off")
	mustFail("completion", "wat")
	mustFail("completion", "--wat")
	mustFail("completion", "bash", "zsh")
}
