package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCompletionParityOutputs(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")

	code, out, errOut := runCLI(t, db, "completion")
	if code != 0 || errOut != "" || !strings.Contains(out, "Available Commands:") || !strings.Contains(out, "bash") || !strings.Contains(out, "Use \"sq completion [command] --help\"") {
		t.Fatalf("completion help failed code=%d out=%q err=%q", code, out, errOut)
	}

	cases := []struct {
		args    []string
		needles []string
	}{
		{[]string{"completion", "bash"}, []string{"# bash completion", "__start_sq", "complete -o default -F __start_sq sq"}},
		{[]string{"completion", "zsh"}, []string{"#compdef sq", "compdef _sq sq", "# zsh completion for sq"}},
		{[]string{"completion", "fish"}, []string{"# fish completion for sq", "complete -c sq -f", "__fish_seen_subcommand_from completion"}},
		{[]string{"completion", "powershell"}, []string{"# powershell completion for sq", "Register-ArgumentCompleter", "-CommandName 'sq'"}},
	}
	for _, tc := range cases {
		code, out, errOut = runCLI(t, db, tc.args...)
		if code != 0 || errOut != "" {
			t.Fatalf("%v failed code=%d out=%q err=%q", tc.args, code, out, errOut)
		}
		for _, needle := range tc.needles {
			if !strings.Contains(out, needle) {
				t.Fatalf("%v missing %q in %q", tc.args, needle, out)
			}
		}
	}
}

func TestCompletionParityHelpAndFlags(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")

	code, out, errOut := runCLI(t, db, "completion", "bash", "--help")
	if code != 0 || errOut != "" || !strings.Contains(out, "help for bash") || !strings.Contains(out, "--no-descriptions") || !strings.Contains(out, "source <(sq completion bash)") {
		t.Fatalf("completion bash --help failed code=%d out=%q err=%q", code, out, errOut)
	}

	code, out, errOut = runCLI(t, db, "completion", "zsh", "--help")
	if code != 0 || errOut != "" || !strings.Contains(out, "help for zsh") || !strings.Contains(out, "source <(sq completion zsh)") {
		t.Fatalf("completion zsh --help failed code=%d out=%q err=%q", code, out, errOut)
	}

	code, out, errOut = runCLI(t, db, "completion", "fish", "--no-descriptions")
	if code != 0 || errOut != "" || !strings.Contains(out, "# fish completion for sq") || !strings.Contains(out, "complete -c sq -f") {
		t.Fatalf("completion fish --no-descriptions failed code=%d out=%q err=%q", code, out, errOut)
	}

	code, _, errOut = runCLI(t, db, "completion", "bash", "--wat")
	if code != 2 || !strings.Contains(errOut, "unknown flag") {
		t.Fatalf("expected unknown flag failure, code=%d err=%q", code, errOut)
	}
}
