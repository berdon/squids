package store

import "testing"

func TestCountTasksAndStatusSummary(t *testing.T) {
	w, done := openTestDB(t)
	defer done()

	alpha, err := CreateTask(w.DB, CreateInput{Title: "alpha", IssueType: "task", Priority: 1})
	if err != nil {
		t.Fatalf("create alpha: %v", err)
	}
	beta, err := CreateTask(w.DB, CreateInput{Title: "beta", IssueType: "bug", Priority: 2})
	if err != nil {
		t.Fatalf("create beta: %v", err)
	}
	gamma, err := CreateTask(w.DB, CreateInput{Title: "gamma", IssueType: "task", Priority: 3})
	if err != nil {
		t.Fatalf("create gamma: %v", err)
	}

	inProgress := "in_progress"
	if _, err := UpdateTask(w.DB, beta.ID, UpdateInput{Status: &inProgress}); err != nil {
		t.Fatalf("update beta: %v", err)
	}
	if _, err := CloseTask(w.DB, gamma.ID, "done"); err != nil {
		t.Fatalf("close gamma: %v", err)
	}
	deferred := "deferred"
	if _, err := UpdateTask(w.DB, alpha.ID, UpdateInput{Status: &deferred}); err != nil {
		t.Fatalf("defer alpha: %v", err)
	}

	all, err := CountTasks(w.DB, "")
	if err != nil {
		t.Fatalf("count all: %v", err)
	}
	if all != 3 {
		t.Fatalf("expected 3 total tasks, got %d", all)
	}

	openCount, err := CountTasks(w.DB, "open")
	if err != nil {
		t.Fatalf("count open: %v", err)
	}
	if openCount != 0 {
		t.Fatalf("expected 0 open tasks, got %d", openCount)
	}

	deferredCount, err := CountTasks(w.DB, "deferred")
	if err != nil {
		t.Fatalf("count deferred: %v", err)
	}
	if deferredCount != 1 {
		t.Fatalf("expected 1 deferred task, got %d", deferredCount)
	}

	summary, err := StatusSummary(w.DB)
	if err != nil {
		t.Fatalf("status summary: %v", err)
	}
	if summary["open"] != 0 || summary["in_progress"] != 1 || summary["closed"] != 1 || summary["deferred"] != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if summary["resolved"] != 0 {
		t.Fatalf("expected resolved default bucket to remain 0, got %+v", summary)
	}
}
