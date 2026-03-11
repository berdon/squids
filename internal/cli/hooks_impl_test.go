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

func TestResolveHooksDir_DefaultGitModeErrorsOutsideGit(t *testing.T) {
	wd := t.TempDir()
	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(wd); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveHooksDir(false, false); err == nil {
		t.Fatal("expected error outside git repo")
	}
}


func TestInjectHookSection_AppendsWhenNoMarkers(t *testing.T) {
	existing := "#!/usr/bin/env sh\necho hi\n"
	out := injectHookSection(existing, generateHookSection("pre-commit"))
	if !strings.Contains(out, "echo hi") || !strings.Contains(out, hookSectionBeginPrefix) {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestInjectHookSection_AppendsWithoutTrailingNewline(t *testing.T) {
	existing := "#!/usr/bin/env sh\necho hi"
	out := injectHookSection(existing, generateHookSection("pre-commit"))
	if !strings.Contains(out, "echo hi") || strings.Count(out, hookSectionBeginPrefix) != 1 {
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

func TestRemoveHookSection(t *testing.T) {
	content := "#!/usr/bin/env sh\n" + generateHookSection("pre-commit") + "echo user\n"
	cleaned, found := removeHookSection(content)
	if !found {
		t.Fatal("expected section found")
	}
	if strings.Contains(cleaned, hookSectionBeginPrefix) || !strings.Contains(cleaned, "echo user") {
		t.Fatalf("unexpected cleaned content: %q", cleaned)
	}
}

func TestUninstallHooksAt_PreservesUserContent(t *testing.T) {
	wd := t.TempDir()
	hooksDir := filepath.Join(wd, ".beads-hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(hooksDir, "pre-commit")
	content := "#!/usr/bin/env sh\n" + generateHookSection("pre-commit") + "echo user\n"
	if err := os.WriteFile(hookPath, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := uninstallHooksAt(hooksDir); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), hookSectionBeginPrefix) || !strings.Contains(string(b), "echo user") {
		t.Fatalf("unexpected remaining hook: %q", string(b))
	}
}

func TestUninstallHooksAt_RemovesEffectivelyEmptyHook(t *testing.T) {
	wd := t.TempDir()
	hooksDir := filepath.Join(wd, ".beads-hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(hookPath, []byte("#!/usr/bin/env sh\n"+generateHookSection("pre-commit")), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := uninstallHooksAt(hooksDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Fatalf("expected hook removed, stat err=%v", err)
	}
}

func TestRemoveHookSection_NotFound(t *testing.T) {
	cleaned, found := removeHookSection("#!/usr/bin/env sh\necho only\n")
	if found || !strings.Contains(cleaned, "echo only") {
		t.Fatalf("unexpected result found=%v content=%q", found, cleaned)
	}
}

func TestRemoveHookSection_BrokenMarkers(t *testing.T) {
	content := hookSectionEndLine() + "\n" + hookSectionBeginLine() + "\n"
	cleaned, found := removeHookSection(content)
	if found || cleaned != content {
		t.Fatalf("expected unchanged for broken markers; found=%v cleaned=%q", found, cleaned)
	}
}

func TestUninstallHooks_NoErrorOutsideGitRepo(t *testing.T) {
	wd := t.TempDir()
	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(wd); err != nil {
		t.Fatal(err)
	}
	if err := uninstallHooks(); err != nil {
		t.Fatalf("unexpected uninstall error: %v", err)
	}
}

func TestUninstallHooksAt_IgnoresMissingFiles(t *testing.T) {
	wd := t.TempDir()
	hooksDir := filepath.Join(wd, ".beads-hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := uninstallHooksAt(hooksDir); err != nil {
		t.Fatalf("expected no error for missing hook files: %v", err)
	}
}

func TestUninstallHooksAt_LeavesUnmanagedHookUntouched(t *testing.T) {
	wd := t.TempDir()
	hooksDir := filepath.Join(wd, ".beads-hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(hookPath, []byte("#!/usr/bin/env sh\necho unmanaged\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := uninstallHooksAt(hooksDir); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "unmanaged") {
		t.Fatalf("expected unmanaged content to stay: %q", string(b))
	}
}

func TestInstallHooks_DefaultGitMode(t *testing.T) {
	wd := t.TempDir()
	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(wd); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v (%s)", err, string(out))
	}
	if err := installHooks(false, false, false); err != nil {
		t.Fatalf("install default git hooks failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(wd, ".git", "hooks", "pre-commit")); err != nil {
		t.Fatalf("expected pre-commit hook: %v", err)
	}
}

func TestUninstallHooks_RemovesManagedFromSharedDir(t *testing.T) {
	wd := t.TempDir()
	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(wd); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(".beads-hooks", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".beads-hooks", "pre-commit"), []byte("#!/usr/bin/env sh\n"+generateHookSection("pre-commit")), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := uninstallHooks(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(".beads-hooks", "pre-commit")); !os.IsNotExist(err) {
		t.Fatalf("expected shared pre-commit removed, got err=%v", err)
	}
}

func TestListHookStatuses_Shared(t *testing.T) {
	wd := t.TempDir()
	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(wd); err != nil {
		t.Fatal(err)
	}
	if err := installHooks(false, true, false); err != nil {
		t.Fatal(err)
	}
	statuses, err := listHookStatuses(true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != len(managedHookNames) {
		t.Fatalf("expected %d statuses, got %d", len(managedHookNames), len(statuses))
	}
	for _, s := range statuses {
		if !s.Installed {
			t.Fatalf("expected installed status for %s", s.Name)
		}
		if !s.IsShim || s.Version == "" {
			t.Fatalf("expected shim+version for %s", s.Name)
		}
	}
}

func TestListHookStatuses_DefaultOutsideGit(t *testing.T) {
	wd := t.TempDir()
	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(wd); err != nil {
		t.Fatal(err)
	}
	statuses, err := listHookStatuses(false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != len(managedHookNames) {
		t.Fatalf("expected %d statuses, got %d", len(managedHookNames), len(statuses))
	}
	for _, s := range statuses {
		if s.Installed {
			t.Fatalf("expected missing status outside git for %s", s.Name)
		}
	}
}
