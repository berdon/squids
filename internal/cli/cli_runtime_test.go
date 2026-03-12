package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func withBadDBPath(t *testing.T, fn func()) {
	t.Helper()
	tmp := t.TempDir()
	badParent := filepath.Join(tmp, "notadir")
	if err := os.WriteFile(badParent, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed bad path: %v", err)
	}
	bad := filepath.Join(badParent, "tasks.sqlite")
	old := os.Getenv("SQ_DB_PATH")
	_ = os.Setenv("SQ_DB_PATH", bad)
	defer func() { _ = os.Setenv("SQ_DB_PATH", old) }()
	fn()
}

func TestCommandRuntimeFailures(t *testing.T) {
	withBadDBPath(t, func() {
		checks := []func() int{
			func() int { return cmdInit() },
			func() int { return cmdReady(nil) },
			func() int { return cmdCreate([]string{"x", "--json"}) },
			func() int { return cmdShow([]string{"bd-1", "--json"}) },
			func() int { return cmdList([]string{"--json"}) },
			func() int { return cmdUpdate([]string{"bd-1", "--json"}) },
			func() int { return cmdClose([]string{"bd-1", "--json"}) },
			func() int { return cmdReopen([]string{"bd-1", "--json"}) },
			func() int { return cmdDelete([]string{"bd-1", "--force", "--json"}) },
			func() int { return cmdLabel([]string{"list-all", "--json"}) },
			func() int { return cmdDep([]string{"list", "bd-1", "--json"}) },
			func() int { return cmdComments([]string{"bd-1", "--json"}) },
			func() int { return cmdTodo([]string{"list", "--json"}) },
			func() int { return cmdChildren([]string{"bd-1", "--json"}) },
			func() int { return cmdBlocked([]string{"--json"}) },
			func() int { return cmdDuplicate([]string{"bd-1", "--of", "bd-2", "--json"}) },
			func() int { return cmdSupersede([]string{"bd-1", "--with", "bd-2", "--json"}) },
			func() int { return cmdQuery([]string{"status=open", "--json"}) },
			func() int { return cmdSearch([]string{"x", "--json"}) },
			func() int { return cmdCount([]string{"--json"}) },
			func() int { return cmdStatus(nil) },
		}
		for i, c := range checks {
			if code := c(); code != 1 {
				t.Fatalf("expected runtime failure code=1 for check %d got %d", i, code)
			}
		}
	})
}
