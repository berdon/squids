package setup_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func requireBash(t *testing.T) string { t.Helper(); p, err := exec.LookPath("bash"); if err != nil { t.Skip("bash not installed") }; return p }
func buildSQBinary(t *testing.T, repoRoot string) string { t.Helper(); binPath := filepath.Join(t.TempDir(), "sq-test-bin"); cmd := exec.Command("go", "build", "-o", binPath, "./cmd/sq"); cmd.Dir = repoRoot; cmd.Env = os.Environ(); out, err := cmd.CombinedOutput(); if err != nil { t.Fatalf("build sq binary: %v\n%s", err, out) }; return binPath }
func TestSetupShellAutomation(t *testing.T) { bash := requireBash(t); repoRoot, err := filepath.Abs(filepath.Clean("../../..")); if err != nil { t.Fatalf("resolve repo root: %v", err) }; sqBin := buildSQBinary(t, repoRoot); scriptPath := filepath.Join(repoRoot, "tests", "shell-automation", "setup", "test.sh"); cmd := exec.Command(bash, scriptPath); cmd.Dir = repoRoot; cmd.Env = append(os.Environ(), "SQ_BIN="+sqBin); out, err := cmd.CombinedOutput(); if err != nil { t.Fatalf("setup shell automation failed: %v\n%s", err, out) } }
