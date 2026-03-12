package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestReadyAdditionalParityFlagsAndFilters(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	if code, _, _ := runCLI(t, db, "init", "--json"); code != 0 {
		t.Fatalf("init failed")
	}

	code, out, _ := runCLI(t, db, "create", "alpha", "--type", "task", "--priority", "1", "--json")
	if code != 0 {
		t.Fatalf("create alpha failed: %q", out)
	}
	alphaID := firstID(t, out)
	if code, _, _ := runCLI(t, db, "update", alphaID, "--add-label", "backend", "--set-metadata", "team=platform", "--json"); code != 0 {
		t.Fatalf("update alpha failed")
	}

	code, out, _ = runCLI(t, db, "create", "beta", "--type", "bug", "--priority", "2", "--json")
	if code != 0 {
		t.Fatalf("create beta failed: %q", out)
	}
	betaID := firstID(t, out)
	if code, _, _ := runCLI(t, db, "update", betaID, "--assignee", "alice", "--add-label", "frontend", "--set-metadata", "team=ui", "--json"); code != 0 {
		t.Fatalf("update beta failed")
	}

	code, out, _ = runCLI(t, db, "ready", "--label-any", "backend,docs", "--has-metadata-key", "team", "--sort", "oldest", "--plain", "--pretty", "--json")
	if code != 0 || !strings.Contains(out, alphaID) || strings.Contains(out, betaID) {
		t.Fatalf("ready label-any/metadata filtering failed code=%d out=%q", code, out)
	}

	code, out, errOut := runCLI(t, db, "ready", "--actor", "tester", "--db", db, "--dolt-auto-commit", "off", "--mol", "ship", "--mol-type", "task", "--rig", "crew", "--include-deferred", "--include-ephemeral", "--gated", "--unassigned", "--json")
	if code != 0 || !strings.Contains(out, alphaID) || strings.Contains(out, betaID) {
		t.Fatalf("ready compat flags failed code=%d out=%q err=%q", code, out, errOut)
	}
}
