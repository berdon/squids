package store

import "testing"

func TestClosedDBErrorPaths(t *testing.T) {
	w, done := openTestDB(t)
	t1, _ := CreateTask(w.DB, CreateInput{Title: "A", IssueType: "task", Priority: 1})
	t2, _ := CreateTask(w.DB, CreateInput{Title: "B", IssueType: "task", Priority: 1})
	done() // close DB to force SQL errors

	if _, err := ListTasks(w.DB); err == nil {
		t.Fatalf("expected list error on closed db")
	}
	if _, err := SearchTasks(w.DB, "A", 1); err == nil {
		t.Fatalf("expected search error on closed db")
	}
	if _, err := CountTasks(w.DB, ""); err == nil {
		t.Fatalf("expected count error on closed db")
	}
	if _, err := StatusSummary(w.DB); err == nil {
		t.Fatalf("expected status error on closed db")
	}
	if err := AddDependency(w.DB, t1.ID, t2.ID, "blocks"); err == nil {
		t.Fatalf("expected add dep error on closed db")
	}
	if _, err := ShowTask(w.DB, t1.ID); err == nil {
		t.Fatalf("expected show error on closed db")
	}
	if _, err := UpdateTask(w.DB, t1.ID, UpdateInput{}); err == nil {
		t.Fatalf("expected update error on closed db")
	}
	if _, err := CloseTask(w.DB, t1.ID, "x"); err == nil {
		t.Fatalf("expected close error on closed db")
	}
	if _, err := ReopenTask(w.DB, t1.ID); err == nil {
		t.Fatalf("expected reopen error on closed db")
	}
	if err := DeleteTask(w.DB, t1.ID); err == nil {
		t.Fatalf("expected delete error on closed db")
	}
	if _, err := RemoveLabel(w.DB, t1.ID, "x"); err == nil {
		t.Fatalf("expected remove label error on closed db")
	}
	if _, err := ListDependencies(w.DB, t1.ID); err == nil {
		t.Fatalf("expected list dependencies error on closed db")
	}
	if err := RemoveDependency(w.DB, t1.ID, t2.ID); err == nil {
		t.Fatalf("expected remove dependency error on closed db")
	}
	if _, err := ListChildren(w.DB, t1.ID); err == nil {
		t.Fatalf("expected list children error on closed db")
	}
	if _, err := ListBlocked(w.DB); err == nil {
		t.Fatalf("expected list blocked error on closed db")
	}
	if _, err := ListComments(w.DB, t1.ID); err == nil {
		t.Fatalf("expected list comments error on closed db")
	}
}
