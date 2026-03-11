package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var managedHookNames = []string{"pre-commit", "post-merge", "pre-push", "post-checkout", "prepare-commit-msg"}

const hookSectionBeginPrefix = "# --- BEGIN SQ INTEGRATION"
const hookSectionEndPrefix = "# --- END SQ INTEGRATION"

func hookSectionBeginLine() string { return hookSectionBeginPrefix + " ---" }
func hookSectionEndLine() string   { return hookSectionEndPrefix + " ---" }

func generateHookSection(hookName string) string {
	return hookSectionBeginLine() + "\n" +
		"# Managed by sq.\n" +
		"if command -v sq >/dev/null 2>&1; then\n" +
		"  sq hooks run " + hookName + " \"$@\"\n" +
		"  _sq_exit=$?; if [ $_sq_exit -ne 0 ]; then exit $_sq_exit; fi\n" +
		"fi\n" +
		hookSectionEndLine() + "\n"
}

func injectHookSection(existing, section string) string {
	beginIdx := strings.Index(existing, hookSectionBeginPrefix)
	endIdx := strings.Index(existing, hookSectionEndPrefix)
	if beginIdx != -1 && endIdx != -1 && beginIdx < endIdx {
		lineStart := strings.LastIndex(existing[:beginIdx], "\n")
		if lineStart == -1 {
			lineStart = 0
		} else {
			lineStart++
		}
		endOfEnd := endIdx + len(hookSectionEndPrefix)
		rest := existing[endOfEnd:]
		if nl := strings.Index(rest, "\n"); nl != -1 {
			endOfEnd += nl + 1
		} else {
			endOfEnd = len(existing)
		}
		return existing[:lineStart] + section + existing[endOfEnd:]
	}
	out := existing
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out + "\n" + section
}

func removeHookSection(content string) (string, bool) {
	beginIdx := strings.Index(content, hookSectionBeginPrefix)
	endIdx := strings.Index(content, hookSectionEndPrefix)
	if beginIdx == -1 || endIdx == -1 || beginIdx > endIdx {
		return content, false
	}
	lineStart := strings.LastIndex(content[:beginIdx], "\n")
	if lineStart == -1 {
		lineStart = 0
	} else {
		lineStart++
	}
	endOfEnd := endIdx + len(hookSectionEndPrefix)
	rest := content[endOfEnd:]
	if nl := strings.Index(rest, "\n"); nl != -1 {
		endOfEnd += nl + 1
	} else {
		endOfEnd = len(content)
	}
	out := content[:lineStart] + content[endOfEnd:]
	for strings.Contains(out, "\n\n\n") {
		out = strings.ReplaceAll(out, "\n\n\n", "\n\n")
	}
	return out, true
}

func installHooks(force, shared, beadsHooks bool) error {
	hooksDir, err := resolveHooksDir(shared, beadsHooks)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return err
	}
	for _, hook := range managedHookNames {
		hookPath := filepath.Join(hooksDir, hook)
		section := generateHookSection(hook)
		existing, err := os.ReadFile(hookPath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		newContent := "#!/usr/bin/env sh\n" + section
		if err == nil {
			existingStr := string(existing)
			if force {
				newContent = "#!/usr/bin/env sh\n" + section
			} else {
				newContent = injectHookSection(existingStr, section)
			}
		}
		if err := os.WriteFile(hookPath, []byte(strings.ReplaceAll(newContent, "\r\n", "\n")), 0o755); err != nil {
			return err
		}
	}
	_ = shared
	_ = beadsHooks
	return nil
}

func resolveHooksDir(shared, beadsHooks bool) (string, error) {
	if beadsHooks {
		dbPath, err := dbPathFromEnvOrCwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(filepath.Dir(dbPath), "hooks"), nil
	}
	if shared {
		return ".beads-hooks", nil
	}
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in git repo: %w", err)
	}
	gitDir := strings.TrimSpace(string(out))
	if !filepath.IsAbs(gitDir) {
		cwd, _ := os.Getwd()
		gitDir = filepath.Join(cwd, gitDir)
	}
	return filepath.Join(gitDir, "hooks"), nil
}

func uninstallHooks() error {
	// remove from default git hooks dir
	if hooksDir, err := resolveHooksDir(false, false); err == nil {
		_ = uninstallHooksAt(hooksDir)
	}
	// remove from shared hooks dir
	_ = uninstallHooksAt(".beads-hooks")
	// remove from beads/sq hooks dir
	if hooksDir, err := resolveHooksDir(false, true); err == nil {
		_ = uninstallHooksAt(hooksDir)
	}
	return nil
}

func uninstallHooksAt(hooksDir string) error {
	for _, hook := range managedHookNames {
		hookPath := filepath.Join(hooksDir, hook)
		b, err := os.ReadFile(hookPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		cleaned, found := removeHookSection(string(b))
		if !found {
			continue
		}
		trim := strings.TrimSpace(cleaned)
		if trim == "" || trim == "#!/usr/bin/env sh" || trim == "#!/bin/sh" {
			if err := os.Remove(hookPath); err != nil && !os.IsNotExist(err) {
				return err
			}
			continue
		}
		if err := os.WriteFile(hookPath, []byte(cleaned), 0o755); err != nil {
			return err
		}
	}
	return nil
}
