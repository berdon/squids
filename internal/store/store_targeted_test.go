package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestRenameTaskAndPrefix(t *testing.T) {
	w, done := openTestDB(t)
	defer done()

	a, _ := CreateTask(w.DB, CreateInput{Title: "A", IssueType: "task", Priority: 1})
	b, _ := CreateTask(w.DB, CreateInput{Title: "B", IssueType: "task", Priority: 1})
	_ = AddDependency(w.DB, a.ID, b.ID, "blocks")
	_, _ = AddComment(w.DB, a.ID, "me", "hello")

	if _, err := RenameTask(w.DB, "", "x"); err == nil {
		t.Fatalf("expected empty old id error")
	}
	if _, err := RenameTask(w.DB, a.ID, a.ID); err == nil {
		t.Fatalf("expected same id error")
	}
	if _, err := RenameTask(w.DB, "bd-missing", "bd-new"); err == nil {
		t.Fatalf("expected missing old id error")
	}
	if _, err := RenameTask(w.DB, a.ID, b.ID); err == nil {
		t.Fatalf("expected new-id exists error")
	}

	renamed, err := RenameTask(w.DB, a.ID, "bd-renamed")
	if err != nil {
		t.Fatalf("rename task: %v", err)
	}
	if renamed.ID != "bd-renamed" {
		t.Fatalf("expected renamed id got %s", renamed.ID)
	}
	if _, err := ShowTask(w.DB, a.ID); err == nil {
		t.Fatalf("old id should be gone")
	}
	if deps, err := ListDependencies(w.DB, "bd-renamed"); err != nil || len(deps) == 0 {
		t.Fatalf("expected deps preserved err=%v deps=%v", err, deps)
	}
	if comments, err := ListComments(w.DB, "bd-renamed"); err != nil || len(comments) == 0 {
		t.Fatalf("expected comments preserved err=%v comments=%v", err, comments)
	}

	if _, err := RenamePrefix(w.DB, "", "sq"); err == nil {
		t.Fatalf("expected empty old prefix error")
	}
	if n, err := RenamePrefix(w.DB, "sq", "sq"); err != nil || n != 0 {
		t.Fatalf("expected no-op rename prefix, got n=%d err=%v", n, err)
	}
	n, err := RenamePrefix(w.DB, "bd", "sq")
	if err != nil {
		t.Fatalf("rename prefix: %v", err)
	}
	if n < 1 {
		t.Fatalf("expected at least one rename, got %d", n)
	}
	if _, err := ShowTask(w.DB, "sq-renamed"); err != nil {
		t.Fatalf("expected sq-renamed to exist: %v", err)
	}
}

func TestStaleAndOrphanTasks(t *testing.T) {
	w, done := openTestDB(t)
	defer done()

	a, _ := CreateTask(w.DB, CreateInput{Title: "stale-candidate", IssueType: "task", Priority: 1})
	// backdate updated_at to mark stale
	oldTs := time.Now().UTC().Add(-40 * 24 * time.Hour).Format(time.RFC3339)
	_, _ = w.DB.Exec(`UPDATE tasks SET updated_at=? WHERE id=?`, oldTs, a.ID)
	stale, err := StaleTasks(w.DB, 30)
	if err != nil {
		t.Fatalf("stale tasks: %v", err)
	}
	found := false
	for _, s := range stale {
		if s.ID == a.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected stale task to be returned")
	}
	// invalid timestamp should be skipped without error
	bad, _ := CreateTask(w.DB, CreateInput{Title: "bad-ts", IssueType: "task", Priority: 1})
	_, _ = w.DB.Exec(`UPDATE tasks SET updated_at=? WHERE id=?`, "not-a-time", bad.ID)
	if _, err := StaleTasks(w.DB, -1); err != nil {
		t.Fatalf("stale tasks default-days path failed: %v", err)
	}

	owner, _ := CreateTask(w.DB, CreateInput{Title: "orphan-owner", IssueType: "task", Priority: 1})
	_, _ = w.DB.Exec(`INSERT INTO dependencies(issue_id,depends_on_id,dep_type) VALUES (?,?,?)`, owner.ID, "bd-missing-ref", "blocks")
	orphans, err := OrphanTasks(w.DB)
	if err != nil {
		t.Fatalf("orphan tasks: %v", err)
	}
	of := false
	for _, o := range orphans {
		if o.ID == owner.ID {
			of = true
		}
	}
	if !of {
		t.Fatalf("expected orphan owner task in results")
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
