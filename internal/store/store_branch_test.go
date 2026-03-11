package store

import "testing"

func TestExecTxError(t *testing.T) {
	w, done := openTestDB(t)
	defer done()
	tx, err := w.DB.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback()
	if err := execTx(tx, "CREATE TABLE IF NOT EXISTS x (id INTEGER)", "ok"); err != nil {
		t.Fatalf("expected execTx success: %v", err)
	}
	if err := execTx(tx, "INVALID SQL", "bad"); err == nil {
		t.Fatalf("expected execTx error")
	}
}

func TestCreateTaskValidationAndConfigFallbacks(t *testing.T) {
	w, done := openTestDB(t)
	defer done()

	if _, err := CreateTask(w.DB, CreateInput{}); err == nil {
		t.Fatalf("expected title required")
	}

	// Force fallback config paths
	_, _ = w.DB.Exec(`DELETE FROM config WHERE key='min_hash_length'`)
	_, _ = w.DB.Exec(`DELETE FROM config WHERE key='max_hash_length'`)
	_, _ = w.DB.Exec(`DELETE FROM config WHERE key='max_collision_prob'`)
	_, _ = w.DB.Exec(`DELETE FROM config WHERE key='id_prefix'`)

	t1, err := CreateTask(w.DB, CreateInput{Title: "fallback", IssueType: "task", Priority: 2})
	if err != nil {
		t.Fatalf("create with fallback config: %v", err)
	}
	if t1.ID == "" {
		t.Fatalf("expected id")
	}

	// Exercise clamp branches
	_, _ = w.DB.Exec(`INSERT OR REPLACE INTO config(key,value) VALUES ('min_hash_length','1')`)
	_, _ = w.DB.Exec(`INSERT OR REPLACE INTO config(key,value) VALUES ('max_hash_length','99')`)
	_, _ = w.DB.Exec(`INSERT OR REPLACE INTO config(key,value) VALUES ('max_collision_prob','0.01')`)
	if _, err := CreateTask(w.DB, CreateInput{Title: "clamp", IssueType: "task", Priority: 1}); err != nil {
		t.Fatalf("create with clamped hash config: %v", err)
	}
}

func TestDependencyAndLabelErrorPaths(t *testing.T) {
	w, done := openTestDB(t)
	defer done()

	a, _ := CreateTask(w.DB, CreateInput{Title: "a", IssueType: "task", Priority: 1})
	b, _ := CreateTask(w.DB, CreateInput{Title: "b", IssueType: "task", Priority: 1})

	if err := AddDependency(w.DB, a.ID, b.ID, ""); err != nil {
		t.Fatalf("default dep type path failed: %v", err)
	}
	if err := AddDependency(w.DB, a.ID, "missing", "blocks"); err == nil {
		t.Fatalf("expected missing dependency target error")
	}
	if err := AddDependency(w.DB, "missing", b.ID, "blocks"); err == nil {
		t.Fatalf("expected missing issue error")
	}
	if _, err := ListLabels(w.DB, "missing"); err == nil {
		t.Fatalf("expected list labels missing issue error")
	}
	if _, err := RemoveLabel(w.DB, "missing", "x"); err == nil {
		t.Fatalf("expected remove label missing issue error")
	}
}

func TestListChildrenAndBlockedEmpty(t *testing.T) {
	w, done := openTestDB(t)
	defer done()

	a, _ := CreateTask(w.DB, CreateInput{Title: "a", IssueType: "epic", Priority: 1})
	// Insert dangling child dependency to exercise missing-child continue path.
	_, _ = w.DB.Exec(`INSERT INTO dependencies(issue_id,depends_on_id,dep_type) VALUES (?,?,?)`, "bd-missing", a.ID, "parent-child")
	children, err := ListChildren(w.DB, a.ID)
	if err != nil || len(children) != 0 {
		t.Fatalf("expected empty children err=%v len=%d", err, len(children))
	}
	blocked, err := ListBlocked(w.DB)
	if err != nil || len(blocked) != 0 {
		t.Fatalf("expected empty blocked err=%v len=%d", err, len(blocked))
	}
}

func TestUpdateTaskClaimAndMetadataInit(t *testing.T) {
	w, done := openTestDB(t)
	defer done()

	a, _ := CreateTask(w.DB, CreateInput{Title: "a", IssueType: "task", Priority: 1})
	b, _ := CreateTask(w.DB, CreateInput{Title: "b", IssueType: "task", Priority: 1})

	up, err := UpdateTask(w.DB, a.ID, UpdateInput{
		Claim:       true,
		SetMetadata: map[string]string{"upstream": b.ID, "": "ignored"},
	})
	if err != nil {
		t.Fatalf("update claim: %v", err)
	}
	if up.Status != "in_progress" {
		t.Fatalf("expected in_progress got %s", up.Status)
	}
	if len(up.Deps) == 0 {
		t.Fatalf("expected upstream dependency in deps")
	}
}

func TestListLabelsRemoveNonExistent(t *testing.T) {
	w, done := openTestDB(t)
	defer done()
	if _, err := ListLabels(w.DB, "bd-missing"); err == nil {
		t.Fatalf("expected list labels missing error")
	}
	if _, err := RemoveLabel(w.DB, "bd-missing", "x"); err == nil {
		t.Fatalf("expected remove label missing error")
	}
}

func TestCloseTaskWithReason(t *testing.T) {
	w, done := openTestDB(t)
	defer done()
	a, _ := CreateTask(w.DB, CreateInput{Title: "a", IssueType: "task", Priority: 1})
	closed, err := CloseTask(w.DB, a.ID, "done now")
	if err != nil {
		t.Fatalf("close task: %v", err)
	}
	if closed.Status != "closed" {
		t.Fatalf("expected closed got %s", closed.Status)
	}
	if closed.CloseReason != "done now" {
		t.Fatalf("expected reason done now got %s", closed.CloseReason)
	}
}
