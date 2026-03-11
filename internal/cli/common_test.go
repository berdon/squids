package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDBPathFromEnvOrCwd(t *testing.T) {
	old := os.Getenv("SQ_DB_PATH")
	defer func() { _ = os.Setenv("SQ_DB_PATH", old) }()

	_ = os.Setenv("SQ_DB_PATH", "/tmp/custom.sqlite")
	p, err := dbPathFromEnvOrCwd()
	if err != nil || p != "/tmp/custom.sqlite" {
		t.Fatalf("env path failed p=%q err=%v", p, err)
	}

	_ = os.Unsetenv("SQ_DB_PATH")
	p, err = dbPathFromEnvOrCwd()
	if err != nil {
		t.Fatalf("cwd path err=%v", err)
	}
	if filepath.Base(p) != "tasks.sqlite" {
		t.Fatalf("unexpected default path: %q", p)
	}
}

func TestOpenTaskDB_BadPath(t *testing.T) {
	tmp := t.TempDir()
	bad := filepath.Join(tmp, "notadir", "tasks.sqlite")
	if err := os.WriteFile(filepath.Join(tmp, "notadir"), []byte("x"), 0o644); err != nil {
		t.Fatalf("seed bad path: %v", err)
	}
	old := os.Getenv("SQ_DB_PATH")
	defer func() { _ = os.Setenv("SQ_DB_PATH", old) }()
	_ = os.Setenv("SQ_DB_PATH", bad)

	if _, _, err := openTaskDB(); err == nil {
		t.Fatalf("expected openTaskDB error")
	}
}
