package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommentsParityBehaviors(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	if code, _, errOut := runCLI(t, db, "init", "--json"); code != 0 {
		t.Fatalf("init failed code=%d err=%q", code, errOut)
	}
	code, out, errOut := runCLI(t, db, "create", "comment target", "--json")
	if code != 0 {
		t.Fatalf("create failed code=%d err=%q", code, errOut)
	}
	id := firstID(t, out)

	code, out, errOut = runCLI(t, db, "comments", id)
	if code != 0 || !strings.Contains(out, "No comments found") {
		t.Fatalf("expected empty human comments output code=%d out=%q err=%q", code, out, errOut)
	}

	commentFile := filepath.Join(t.TempDir(), "comment.txt")
	if err := os.WriteFile(commentFile, []byte("comment from file\n"), 0o644); err != nil {
		t.Fatalf("write comment file: %v", err)
	}
	code, out, errOut = runCLI(t, db, "comments", "add", id, "-f", commentFile, "--author", "alice")
	if code != 0 || !strings.Contains(out, "Added comment") {
		t.Fatalf("expected human add output code=%d out=%q err=%q", code, out, errOut)
	}

	code, out, errOut = runCLI(t, db, "comments", id, "--json")
	if code != 0 {
		t.Fatalf("comments list json failed code=%d err=%q", code, errOut)
	}
	var comments []map[string]any
	if err := json.Unmarshal([]byte(out), &comments); err != nil {
		t.Fatalf("decode comments json: %v payload=%q", err, out)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment got %d payload=%q", len(comments), out)
	}
	if comments[0]["author"] != "alice" {
		t.Fatalf("expected author alice payload=%q", out)
	}
	if !strings.Contains(comments[0]["body"].(string), "comment from file") {
		t.Fatalf("expected file body payload=%q", out)
	}

	code, out, errOut = runCLI(t, db, "comments", id, "--local-time")
	if code != 0 || !strings.Contains(out, "alice") || !strings.Contains(out, "comment from file") {
		t.Fatalf("expected local-time human output code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestCommentsParityValidationErrors(t *testing.T) {
	db := filepath.Join(t.TempDir(), "tasks.sqlite")
	_, _, _ = runCLI(t, db, "init", "--json")
	id := firstID(t, mustRunCLI(t, db, "create", "comment target", "--json"))

	mustFailCLI := func(args ...string) {
		t.Helper()
		code, _, _ := runCLI(t, db, args...)
		if code == 0 {
			t.Fatalf("expected failure for %v", args)
		}
	}

	mustFailCLI("comments", id, "--wat")
	mustFailCLI("comments", "add", id, "--wat")
	mustFailCLI("comments", "add", id, "-f")
	mustFailCLI("comments", "add", id, "--author")
	mustFailCLI("comments", "add", id, "-f", filepath.Join(t.TempDir(), "missing.txt"))
	mustFailCLI("comments", "add", "--db")
	mustFailCLI("comments", "--db")
}

func TestFormatCommentTimeFallsBackOnInvalidTimestamp(t *testing.T) {
	const bad = "not-a-time"
	if got := formatCommentTime(bad, false); got != bad {
		t.Fatalf("expected unchanged time when local=false got %q", got)
	}
	if got := formatCommentTime(bad, true); got != bad {
		t.Fatalf("expected unchanged invalid time got %q", got)
	}
}

func mustRunCLI(t *testing.T, db string, args ...string) string {
	t.Helper()
	code, out, errOut := runCLI(t, db, args...)
	if code != 0 {
		t.Fatalf("runCLI failed args=%v code=%d err=%q", args, code, errOut)
	}
	return out
}
