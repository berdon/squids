package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCompatibilityCommandsMissingValueFlags(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	_, _, _ = runCLI(t, db, "init", "--json")

	cases := [][]string{
		{"comments", "add", "--actor"},
		{"comments", "add", "--db"},
		{"comments", "add", "--dolt-auto-commit"},
		{"comments", "--actor"},
		{"comments", "--db"},
		{"comments", "--dolt-auto-commit"},
		{"edit", "--actor"},
		{"edit", "--db"},
		{"edit", "--dolt-auto-commit"},
		{"dolt", "--actor"},
		{"dolt", "--db"},
		{"dolt", "--dolt-auto-commit"},
		{"linear", "--actor"},
		{"linear", "--db"},
		{"linear", "--dolt-auto-commit"},
		{"memories", "--actor"},
		{"memories", "--db"},
		{"memories", "--dolt-auto-commit"},
	}
	for _, tc := range cases {
		code, _, errOut := runCLI(t, db, tc...)
		if code == 0 {
			t.Fatalf("expected failure for %v", tc)
		}
		if !strings.Contains(errOut, "missing value") && !strings.Contains(errOut, "unknown flag") {
			t.Fatalf("expected flag parsing failure for %v err=%q", tc, errOut)
		}
	}
}
