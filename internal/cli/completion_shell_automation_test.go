package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func requireShell(t *testing.T, shell string) string {
	t.Helper()
	path, err := exec.LookPath(shell)
	if err != nil {
		t.Skipf("%s not installed", shell)
	}
	return path
}

func buildTestSQBinary(t *testing.T) string {
	t.Helper()
	repoRoot := filepath.Clean("../..")
	binPath := filepath.Join(t.TempDir(), "sq-test-bin")
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/sq")
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build sq test binary failed: %v\n%s", err, out)
	}
	return binPath
}

func TestCompletionShellAutomation_Bash(t *testing.T) {
	bash := requireShell(t, "bash")
	sqBin := buildTestSQBinary(t)

	script := `source <($SQ_BIN completion bash)
COMP_WORDS=(sq up)
COMP_CWORD=1
__start_sq
printf '%s
' "${COMPREPLY[@]}"
`
	cmd := exec.Command(bash, "--noprofile", "--norc", "-c", script)
	cmd.Dir = filepath.Clean("../..")
	cmd.Env = append(os.Environ(), "SQ_BIN="+sqBin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bash completion failed: %v\n%s", err, out)
	}
	result := string(out)
	if !strings.Contains(result, "update") {
		t.Fatalf("expected bash completion to suggest update, got %q", result)
	}

	script = `source <($SQ_BIN completion bash)
COMP_WORDS=(sq completion z)
COMP_CWORD=2
__start_sq
printf '%s
' "${COMPREPLY[@]}"
`
	cmd = exec.Command(bash, "--noprofile", "--norc", "-c", script)
	cmd.Dir = filepath.Clean("../..")
	cmd.Env = append(os.Environ(), "SQ_BIN="+sqBin)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bash completion subcommand failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "zsh") {
		t.Fatalf("expected bash completion to suggest zsh, got %q", string(out))
	}
}

func TestCompletionShellAutomation_Zsh(t *testing.T) {
	zsh := requireShell(t, "zsh")
	sqBin := buildTestSQBinary(t)
	script := `autoload -Uz compinit && compinit
source <($SQ_BIN completion zsh)
compadd() { shift; printf '%s
' "$@"; }
words=(sq up)
CURRENT=2
_sq
words=(sq completion z)
CURRENT=3
_sq
`
	cmd := exec.Command(zsh, "-f", "-c", script)
	cmd.Dir = filepath.Clean("../..")
	cmd.Env = append(os.Environ(), "SQ_BIN="+sqBin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("zsh completion failed: %v\n%s", err, out)
	}
	result := string(out)
	if !strings.Contains(result, "update") || !strings.Contains(result, "zsh") {
		t.Fatalf("expected zsh completion suggestions, got %q", result)
	}
	if strings.Contains(result, "Update a task") || strings.Contains(result, "Generate shell completion scripts") {
		t.Fatalf("expected raw completion values without descriptions, got %q", result)
	}
}

func TestCompletionShellAutomation_Fish(t *testing.T) {
	fish := requireShell(t, "fish")
	sqBin := buildTestSQBinary(t)
	script := `source ($SQ_BIN completion fish | psub)
complete --do-complete "sq up"
complete --do-complete "sq completion z"
`
	cmd := exec.Command(fish, "-c", script)
	cmd.Dir = filepath.Clean("../..")
	cmd.Env = append(os.Environ(), "SQ_BIN="+sqBin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fish completion failed: %v\n%s", err, out)
	}
	result := string(out)
	if !strings.Contains(result, "update") || !strings.Contains(result, "zsh") {
		t.Fatalf("expected fish completion suggestions, got %q", result)
	}
}
