package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withEnv(key, value string, fn func()) {
	old := os.Getenv(key)
	_ = os.Setenv(key, value)
	defer func() { _ = os.Setenv(key, old) }()
	fn()
}

func TestCmdBackupAndGate_BranchCoverage(t *testing.T) {
	tmp := t.TempDir()
	db := filepath.Join(tmp, "tasks.sqlite")

	withEnv("SQ_DB_PATH", db, func() {
		if code := cmdInit(); code != 0 {
			t.Fatalf("init failed code=%d", code)
		}
		if code := cmdCreate([]string{"gate task", "--type", "gate", "--json"}); code != 0 {
			t.Fatalf("create gate failed code=%d", code)
		}

		// backup --json (flag-first -> export branch)
		if code := cmdBackup([]string{"--json"}); code != 0 {
			t.Fatalf("backup --json failed code=%d", code)
		}
		// backup status (human)
		if code := cmdBackup([]string{"status"}); code != 0 {
			t.Fatalf("backup status failed code=%d", code)
		}
		// unsupported compat branches
		if code := cmdBackup([]string{"init", "/tmp/backup-remote"}); code == 0 {
			t.Fatalf("backup init expected unsupported non-zero")
		}
		if code := cmdBackup([]string{"sync"}); code == 0 {
			t.Fatalf("backup sync expected unsupported non-zero")
		}
		if code := cmdBackup([]string{"wat"}); code == 0 {
			t.Fatalf("backup unknown expected non-zero")
		}

		// gate help + unknown branch
		if code := cmdGate([]string{"--help"}); code != 0 {
			t.Fatalf("gate --help failed code=%d", code)
		}
		if code := cmdGate([]string{"wat"}); code == 0 {
			t.Fatalf("gate unknown expected non-zero")
		}
	})
}

func TestCmdBackup_RestoreMissingPathFails(t *testing.T) {
	tmp := t.TempDir()
	db := filepath.Join(tmp, "tasks.sqlite")
	withEnv("SQ_DB_PATH", db, func() {
		if code := cmdInit(); code != 0 {
			t.Fatalf("init failed code=%d", code)
		}
		code, _, errOut := captureOutput(t, func() int { return cmdBackup([]string{"restore", "/definitely/missing.sqlite", "--json"}) })
		if code == 0 || !strings.Contains(strings.ToLower(errOut), "failed") {
			t.Fatalf("expected restore missing-path failure code=%d err=%q", code, errOut)
		}
	})
}
