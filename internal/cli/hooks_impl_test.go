package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInjectHookSection_ReplacesExistingManagedBlock(t *testing.T) {
	existing := "#!/usr/bin/env sh\n" +
		hookSectionBeginLine() + "\nold\n" + hookSectionEndLine() + "\n" +
		"echo user\n"
	section := generateHookSection("pre-commit")
	out := injectHookSection(existing, section)
	if strings.Count(out, hookSectionBeginPrefix) != 1 {
		t.Fatalf("expected one begin marker, got: %q", out)
	}
	if !strings.Contains(out, "echo user") {
		t.Fatalf("expected user content preserved: %q", out)
	}
}

func TestInstallHooks_SharedWritesManagedSection(t *testing.T) {
	wd := t.TempDir()
	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(wd); err != nil {
		t.Fatal(err)
	}
	if err := installHooks(false, true, false); err != nil {
		t.Fatalf("installHooks failed: %v", err)
	}
	for _, h := range managedHookNames {
		p := filepath.Join(wd, ".beads-hooks", h)
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("missing hook %s: %v", h, err)
		}
		s := string(b)
		if !strings.Contains(s, hookSectionBeginPrefix) || !strings.Contains(s, hookSectionEndPrefix) {
			t.Fatalf("hook missing markers: %s", h)
		}
	}
	// idempotent reinstall should keep exactly one section marker block
	if err := installHooks(false, true, false); err != nil {
		t.Fatalf("reinstall failed: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(wd, ".beads-hooks", "pre-commit"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(b), hookSectionBeginPrefix) != 1 {
		t.Fatalf("expected one managed section, got: %q", string(b))
	}
}

func TestResolveHooksDir_BeadsMode(t *testing.T) {
	wd := t.TempDir()
	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(wd); err != nil {
		t.Fatal(err)
	}
	_ = os.MkdirAll(filepath.Join(wd, ".sq"), 0o755)
	p, err := resolveHooksDir(false, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(p, ".sq/hooks") {
		t.Fatalf("unexpected hooks path: %s", p)
	}
}

func TestResolveHooksDir_DefaultGitMode(t *testing.T) {
	wd := t.TempDir()
	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(wd); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v (%s)", err, string(out))
	}
	p, err := resolveHooksDir(false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(p, ".git/hooks") {
		t.Fatalf("unexpected hooks path: %s", p)
	}
}

func TestSetCoreHooksPath_NoGitRepoNoop(t *testing.T) {
	wd := t.TempDir()
	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(wd); err != nil {
		t.Fatal(err)
	}
	if err := setCoreHooksPath(".beads-hooks"); err != nil {
		t.Fatalf("expected noop outside git repo, got error: %v", err)
	}
}

func TestSetCoreHooksPath_GitRepo(t *testing.T) {
	wd := t.TempDir()
	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(wd); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v (%s)", err, string(out))
	}
	if err := setCoreHooksPath(".beads-hooks"); err != nil {
		t.Fatalf("setCoreHooksPath failed: %v", err)
	}
	out, err := exec.Command("git", "config", "--get", "core.hooksPath").CombinedOutput()
	if err != nil {
		t.Fatalf("git config get failed: %v (%s)", err, string(out))
	}
	if strings.TrimSpace(string(out)) != ".beads-hooks" {
		t.Fatalf("unexpected hooksPath: %q", strings.TrimSpace(string(out)))
	}
}

func TestInjectHookSection_AppendsWhenNoMarkers(t *testing.T) {
	existing := "#!/usr/bin/env sh\necho hi\n"
	out := injectHookSection(existing, generateHookSection("pre-commit"))
	if !strings.Contains(out, "echo hi") || !strings.Contains(out, hookSectionBeginPrefix) {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestInstallHooks_ForceReplacesExistingFile(t *testing.T) {
	wd := t.TempDir()
	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(wd); err != nil {
		t.Fatal(err)
	}
	_ = os.MkdirAll(filepath.Join(wd, ".beads-hooks"), 0o755)
	hookPath := filepath.Join(wd, ".beads-hooks", "pre-commit")
	if err := os.WriteFile(hookPath, []byte("#!/usr/bin/env sh\necho legacy\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := installHooks(true, true, false); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "legacy") || !strings.Contains(string(b), hookSectionBeginPrefix) {
		t.Fatalf("force install did not replace properly: %q", string(b))
	}
}
