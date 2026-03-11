package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreOpenBranches(t *testing.T) {
	if _, err := Open(""); err == nil {
		t.Fatalf("expected empty path error")
	}
	tmp := t.TempDir()
	badParent := filepath.Join(tmp, "notadir")
	if err := os.WriteFile(badParent, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed bad parent: %v", err)
	}
	if _, err := Open(filepath.Join(badParent, "db.sqlite")); err == nil {
		t.Fatalf("expected mkdir failure")
	}
	good := filepath.Join(tmp, "ok", "db.sqlite")
	db, err := Open(good)
	if err != nil {
		t.Fatalf("expected open success: %v", err)
	}
	_ = db.Close()
}

func TestCloseReopenAndCommentErrorBranches(t *testing.T) {
	w, done := openTestDB(t)
	defer done()

	if _, err := CloseTask(w.DB, "bd-missing", "x"); err == nil {
		t.Fatalf("expected close missing error")
	}
	if _, err := ReopenTask(w.DB, "bd-missing"); err == nil {
		t.Fatalf("expected reopen missing error")
	}
	if _, err := AddComment(w.DB, "bd-missing", "me", "hello"); err == nil {
		t.Fatalf("expected add comment missing issue error")
	}
	base, _ := CreateTask(w.DB, CreateInput{Title: "c", IssueType: "task", Priority: 1})
	if _, err := AddComment(w.DB, base.ID, "me", "   "); err == nil {
		t.Fatalf("expected empty comment error")
	}
	if _, err := ListComments(w.DB, "bd-missing"); err == nil {
		t.Fatalf("expected list comments missing error")
	}
}

func TestUpdateAndDepsBranches(t *testing.T) {
	w, done := openTestDB(t)
	defer done()

	a, _ := CreateTask(w.DB, CreateInput{Title: "A", IssueType: "task", Priority: 1})
	b, _ := CreateTask(w.DB, CreateInput{Title: "B", IssueType: "task", Priority: 1})
	status := "in_progress"
	assignee := "alice"
	up, err := UpdateTask(w.DB, a.ID, UpdateInput{
		Status:    &status,
		Assignee:  &assignee,
		AddLabels: []string{"x", "x"},
		SetMetadata: map[string]string{
			"upstream": b.ID,
		},
	})
	if err != nil {
		t.Fatalf("update task: %v", err)
	}
	if up.Assignee != "alice" || up.Status != "in_progress" {
		t.Fatalf("unexpected update: %+v", up)
	}
	if len(up.Labels) != 1 {
		t.Fatalf("expected deduped labels got %v", up.Labels)
	}
	if len(up.Deps) == 0 {
		t.Fatalf("expected upstream dep")
	}
	if err := AddDependency(w.DB, a.ID, b.ID, ""); err != nil {
		t.Fatalf("default dep type should work: %v", err)
	}
	deps, err := ListDependencies(w.DB, a.ID)
	if err != nil || len(deps) == 0 {
		t.Fatalf("expected deps err=%v deps=%v", err, deps)
	}
}

func TestReadyTasksFiltersBlocked(t *testing.T) {
	w, done := openTestDB(t)
	defer done()

	blockedTarget, _ := CreateTask(w.DB, CreateInput{Title: "blocked target", IssueType: "task", Priority: 1})
	blocker, _ := CreateTask(w.DB, CreateInput{Title: "blocker", IssueType: "task", Priority: 1})
	openFree, _ := CreateTask(w.DB, CreateInput{Title: "open free", IssueType: "task", Priority: 1})
	closedTask, _ := CreateTask(w.DB, CreateInput{Title: "closed", IssueType: "task", Priority: 1})
	_, _ = CloseTask(w.DB, closedTask.ID, "done")
	if err := AddDependency(w.DB, blocker.ID, blockedTarget.ID, "blocks"); err != nil {
		t.Fatalf("add blocks dep: %v", err)
	}

	ready, err := ReadyTasks(w.DB)
	if err != nil {
		t.Fatalf("ready tasks: %v", err)
	}
	ids := map[string]bool{}
	for _, r := range ready {
		ids[r.ID] = true
	}
	if !ids[blockedTarget.ID] {
		t.Fatalf("blocked target should be ready by bd semantics")
	}
	if ids[blocker.ID] {
		t.Fatalf("blocker (with outgoing blocks dep) should not be ready")
	}
	if !ids[openFree.ID] {
		t.Fatalf("expected open free task to be ready")
	}
	if ids[closedTask.ID] {
		t.Fatalf("closed task should not be ready")
	}
}
