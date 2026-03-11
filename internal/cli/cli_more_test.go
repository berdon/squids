package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRun_UnknownSubcommands(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	_, _, _ = runCLI(t, db, "init", "--json")

	cases := [][]string{
		{"label", "wat"},
		{"dep", "wat"},
	}
	for _, c := range cases {
		code, _, _ := runCLI(t, db, c...)
		if code == 0 {
			t.Fatalf("expected failure for %v", c)
		}
	}
}

func TestPrintJSON_ErrorPath(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	_ = r.Close()
	_ = w.Close()
	os.Stdout = w
	defer func() { os.Stdout = old }()

	if code := printJSON(map[string]any{"x": 1}); code == 0 {
		t.Fatalf("expected printJSON failure code")
	}
}
