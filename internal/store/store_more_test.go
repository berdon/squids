package store

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenAndInitErrors(t *testing.T) {
	if _, err := Open(""); err == nil {
		t.Fatalf("expected error for empty db path")
	}
	if err := Init(nil); err == nil {
		t.Fatalf("expected error for nil db")
	}
	if err := EnsureInitialized(nil); err == nil {
		t.Fatalf("expected error for nil db")
	}
	if _, err := CurrentVersion(nil); err == nil {
		t.Fatalf("expected error for nil db")
	}
	if err := Ping(nil); err == nil {
		t.Fatalf("expected error for nil db")
	}
}

func TestCounterIDModeAndLifecycle(t *testing.T) {
	w, done := openTestDB(t)
	defer done()

	if _, err := w.DB.Exec(`INSERT OR REPLACE INTO config(key,value) VALUES ('issue_id_mode','counter')`); err != nil {
		t.Fatalf("set counter mode: %v", err)
	}

	a, err := CreateTask(w.DB, CreateInput{Title: "A", IssueType: "task", Priority: 1})
	if err != nil {
		t.Fatalf("create a: %v", err)
	}
	b, err := CreateTask(w.DB, CreateInput{Title: "B", IssueType: "task", Priority: 2})
	if err != nil {
		t.Fatalf("create b: %v", err)
	}
	if a.ID == b.ID {
		t.Fatalf("expected unique ids")
	}

	if _, err := UpdateTask(w.DB, a.ID, UpdateInput{Claim: true}); err != nil {
		t.Fatalf("claim update: %v", err)
	}
	if _, err := CloseTask(w.DB, a.ID, "done"); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, err := ReopenTask(w.DB, a.ID); err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if err := DeleteTask(w.DB, b.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := DeleteTask(w.DB, b.ID); err == nil {
		t.Fatalf("expected delete missing to fail")
	}
}

func TestLabelsAndDependencyHelpers(t *testing.T) {
	w, done := openTestDB(t)
	defer done()

	t1, _ := CreateTask(w.DB, CreateInput{Title: "L1", IssueType: "task", Priority: 1})
	t2, _ := CreateTask(w.DB, CreateInput{Title: "L2", IssueType: "task", Priority: 1})

	if _, err := AddLabel(w.DB, t1.ID, "triage"); err != nil {
		t.Fatalf("add label: %v", err)
	}
	if _, err := AddLabel(w.DB, t1.ID, ""); err == nil {
		t.Fatalf("expected empty label error")
	}
	if _, err := RemoveLabel(w.DB, t1.ID, "triage"); err != nil {
		t.Fatalf("remove label: %v", err)
	}
	if labels, err := ListLabels(w.DB, t1.ID); err != nil || len(labels) != 0 {
		t.Fatalf("list labels err=%v labels=%v", err, labels)
	}
	if _, err := AddLabel(w.DB, t2.ID, "type:bug"); err != nil {
		t.Fatalf("add label2: %v", err)
	}
	if all, err := ListAllLabels(w.DB); err != nil || len(all) == 0 {
		t.Fatalf("list all labels err=%v all=%v", err, all)
	}

	if err := AddDependency(w.DB, t1.ID, t2.ID, "blocks"); err != nil {
		t.Fatalf("add dep: %v", err)
	}
	if deps, err := ListDependencies(w.DB, t1.ID); err != nil || len(deps) != 1 {
		t.Fatalf("list deps err=%v deps=%v", err, deps)
	}
	if err := AddDependency(w.DB, "missing", t2.ID, "blocks"); err == nil {
		t.Fatalf("expected missing issue error")
	}
}

func TestDependenciesChildrenBlockedAndComments(t *testing.T) {
	w, done := openTestDB(t)
	defer done()

	parent, _ := CreateTask(w.DB, CreateInput{Title: "Parent", IssueType: "epic", Priority: 1})
	child, _ := CreateTask(w.DB, CreateInput{Title: "Child", IssueType: "task", Priority: 2})
	blocker, _ := CreateTask(w.DB, CreateInput{Title: "Blocker", IssueType: "task", Priority: 1})

	if err := AddDependency(w.DB, child.ID, parent.ID, "parent-child"); err != nil {
		t.Fatalf("parent-child: %v", err)
	}
	if err := AddDependency(w.DB, blocker.ID, child.ID, "blocks"); err != nil {
		t.Fatalf("blocks: %v", err)
	}

	kids, err := ListChildren(w.DB, parent.ID)
	if err != nil || len(kids) == 0 {
		t.Fatalf("children err=%v len=%d", err, len(kids))
	}

	blocked, err := ListBlocked(w.DB)
	if err != nil || len(blocked) == 0 {
		t.Fatalf("blocked err=%v len=%d", err, len(blocked))
	}

	if err := RemoveDependency(w.DB, blocker.ID, child.ID); err != nil {
		t.Fatalf("remove dep: %v", err)
	}
	if err := RemoveDependency(w.DB, blocker.ID, child.ID); err == nil {
		t.Fatalf("expected missing dep remove failure")
	}

	if _, err := AddComment(w.DB, child.ID, "me", "note"); err != nil {
		t.Fatalf("add comment: %v", err)
	}
	if _, err := AddComment(w.DB, child.ID, "me", "   "); err == nil {
		t.Fatalf("expected empty comment failure")
	}
	comments, err := ListComments(w.DB, child.ID)
	if err != nil || len(comments) == 0 {
		t.Fatalf("list comments err=%v len=%d", err, len(comments))
	}
}

func TestSearchCountStatusQueryCoverage(t *testing.T) {
	w, done := openTestDB(t)
	defer done()

	a, _ := CreateTask(w.DB, CreateInput{Title: "Alpha", IssueType: "task", Priority: 1})
	b, _ := CreateTask(w.DB, CreateInput{Title: "Beta", IssueType: "bug", Priority: 3})
	_, _ = UpdateTask(w.DB, a.ID, UpdateInput{Status: strPtr("in_progress")})
	_, _ = CloseTask(w.DB, b.ID, "done")

	if got, err := SearchTasks(w.DB, "alp", 1); err != nil || len(got) != 1 {
		t.Fatalf("search err=%v len=%d", err, len(got))
	}
	if _, err := CountTasks(w.DB, ""); err != nil {
		t.Fatalf("count all: %v", err)
	}
	if _, err := CountTasks(w.DB, "closed"); err != nil {
		t.Fatalf("count status: %v", err)
	}
	if _, err := StatusSummary(w.DB); err != nil {
		t.Fatalf("status summary: %v", err)
	}

	queries := []string{
		"status=closed",
		"type=task",
		"assignee=",
		"title=alp",
		"priority>=1",
		"priority<=3",
		"priority>0",
		"priority<5",
	}
	for _, q := range queries {
		if _, err := QueryTasks(w.DB, q); err != nil {
			t.Fatalf("query %q failed: %v", q, err)
		}
	}
	if _, err := QueryTasks(w.DB, ""); err == nil {
		t.Fatalf("expected empty query error")
	}
	if _, err := QueryTasks(w.DB, "madeup=1"); err == nil {
		t.Fatalf("expected unknown field error")
	}
}

func TestOpenWithNestedPath(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "a", "b", "c.sqlite")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("open nested: %v", err)
	}
	defer db.Close()
	if err := Init(db); err != nil {
		t.Fatalf("init nested: %v", err)
	}
}

func strPtr(s string) *string { return &s }

func TestCurrentVersionScanErrorPath(t *testing.T) {
	// Cover scan error path by using an uninitialized in-memory DB.
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open mem: %v", err)
	}
	defer db.Close()
	if _, err := CurrentVersion(db); err == nil {
		t.Fatalf("expected version scan error")
	}
}
