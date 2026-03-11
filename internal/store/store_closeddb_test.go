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
}
