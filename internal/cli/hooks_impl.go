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
	if shared {
		return setCoreHooksPath(".beads-hooks")
	}
	if beadsHooks {
		return setCoreHooksPath(".sq/hooks")
	}
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

func setCoreHooksPath(path string) error {
	check := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := check.Run(); err != nil {
		return nil
	}
	cmd := exec.Command("git", "config", "core.hooksPath", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git config core.hooksPath failed: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
