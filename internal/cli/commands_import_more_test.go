package cli

import (
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/berdon/squids/internal/store"
)

func TestCmdMemoriesAndGitLabCompatibilityBranches(t *testing.T) {
	code, out, errOut := captureOutput(t, func() int { return cmdMemories([]string{"--help"}) })
	if code != 0 || errOut != "" || !strings.Contains(strings.ToLower(out), "memories") || !strings.Contains(out, "Usage:") {
		t.Fatalf("expected memories help success, code=%d out=%q err=%q", code, out, errOut)
	}

	code, out, _ = captureOutput(t, func() int { return cmdGitLab(nil) })
	if code != 0 || !strings.Contains(out, "sq gitlab") {
		t.Fatalf("expected gitlab usage, code=%d out=%q", code, out)
	}

	code, _, errOut = captureOutput(t, func() int { return cmdGitLab([]string{"projects"}) })
	if code == 0 || !strings.Contains(strings.ToLower(errOut), "not yet supported") {
		t.Fatalf("expected gitlab projects runtime failure, code=%d err=%q", code, errOut)
	}

	code, _, errOut = captureOutput(t, func() int { return cmdGitLab([]string{"wat"}) })
	if code == 0 || !strings.Contains(strings.ToLower(errOut), "unknown gitlab subcommand") {
		t.Fatalf("expected unknown gitlab subcommand failure, code=%d err=%q", code, errOut)
	}
}

func TestParseImportOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		ok   bool
	}{
		{"empty", nil, true},
		{"all flags", []string{"--source", "/tmp/x.db", "--dry-run", "--json", "--no-comments", "--no-events"}, true},
		{"missing source value", []string{"--source"}, false},
		{"unknown", []string{"--wat"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseImportOptions(tt.args)
			if tt.ok && err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestNormalizeSourcePath(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "x.sqlite")
	if err := os.WriteFile(f, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := normalizeSourcePath(f)
	if err != nil || got != f {
		t.Fatalf("normalize file: got=%q err=%v", got, err)
	}

	dir1 := filepath.Join(tmp, "d1")
	_ = os.MkdirAll(dir1, 0o755)
	if err := os.WriteFile(filepath.Join(dir1, "tasks.sqlite"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = normalizeSourcePath(dir1)
	if err != nil || got != filepath.Join(dir1, "tasks.sqlite") {
		t.Fatalf("normalize known db in dir: got=%q err=%v", got, err)
	}

	dir2 := filepath.Join(tmp, "d2")
	_ = os.MkdirAll(dir2, 0o755)
	if err := os.WriteFile(filepath.Join(dir2, "a.db"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = normalizeSourcePath(dir2)
	if err != nil || got != filepath.Join(dir2, "a.db") {
		t.Fatalf("normalize single *.db in dir: got=%q err=%v", got, err)
	}

	dir3 := filepath.Join(tmp, "d3")
	_ = os.MkdirAll(dir3, 0o755)
	_ = os.WriteFile(filepath.Join(dir3, "a.db"), []byte(""), 0o644)
	_ = os.WriteFile(filepath.Join(dir3, "b.sqlite"), []byte(""), 0o644)
	if _, err := normalizeSourcePath(dir3); err == nil {
		t.Fatalf("expected ambiguous error")
	}
}

func TestResolveImportSourceViaEnvAndCwd(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "source.sqlite")
	seedSourceDB(t, src)

	oldDB := os.Getenv("BEADS_DATABASE")
	oldDir := os.Getenv("BEADS_DIR")
	defer func() {
		_ = os.Setenv("BEADS_DATABASE", oldDB)
		_ = os.Setenv("BEADS_DIR", oldDir)
	}()

	_ = os.Setenv("BEADS_DATABASE", src)
	_ = os.Setenv("BEADS_DIR", "")
	got, err := resolveImportSource("")
	if err != nil || got != src {
		t.Fatalf("resolve via env database failed: got=%q err=%v", got, err)
	}

	_ = os.Setenv("BEADS_DATABASE", "")
	_ = os.Setenv("BEADS_DIR", tmp)
	got, err = resolveImportSource("")
	if err != nil || got != src {
		t.Fatalf("resolve via env dir failed: got=%q err=%v", got, err)
	}
}

func TestValidateSourceSchema(t *testing.T) {
	tmp := t.TempDir()
	good := filepath.Join(tmp, "good.sqlite")
	seedSourceDB(t, good)
	db, err := openSQLite(good)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := validateSourceSchema(db); err != nil {
		t.Fatalf("expected valid schema: %v", err)
	}
}

func TestImportFromSource_NoCommentsAndInvalidDeps(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "source.sqlite")
	target := filepath.Join(tmp, "target.sqlite")
	seedSourceDB(t, source)

	sdb, err := store.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	defer sdb.Close()
	_, _ = sdb.Exec(`INSERT INTO tasks(id,title,status,priority,issue_type,created_at,updated_at) VALUES('bd-2','two','open',2,'task','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`)
	_, _ = sdb.Exec(`INSERT INTO dependencies(issue_id,depends_on_id,dep_type) VALUES('bd-1','bd-1','blocks')`)
	_, _ = sdb.Exec(`INSERT INTO dependencies(issue_id,depends_on_id,dep_type) VALUES('bd-1','bd-2','blocks')`)

	tdb, err := store.Open(target)
	if err != nil {
		t.Fatal(err)
	}
	defer tdb.Close()
	if err := store.Init(tdb); err != nil {
		t.Fatal(err)
	}

	report, err := importFromSource(sdb, tdb, source, importOptions{NoComments: true})
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if report.Tasks["created"] < 2 {
		t.Fatalf("expected tasks created, got %+v", report.Tasks)
	}
	if report.Comments["created"] != 0 {
		t.Fatalf("expected no comments created, got %+v", report.Comments)
	}
	if len(report.Warnings) == 0 {
		t.Fatalf("expected warning for invalid self-dependency")
	}

	row := tdb.QueryRow(`SELECT COUNT(1) FROM dependencies WHERE issue_id='bd-1' AND depends_on_id='bd-2'`)
	var n int
	if err := row.Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected valid dependency imported")
	}
}

func TestResolveImportSourceAmbiguous(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "beads")
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "one.db"), []byte(""), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "two.sqlite"), []byte(""), 0o644)
	if _, err := resolveImportSource(dir); err == nil {
		t.Fatalf("expected ambiguity error")
	}
}

func TestOpenSQLiteFailure(t *testing.T) {
	if _, err := openSQLite("/definitely/missing/path/nope.sqlite"); err == nil {
		t.Fatalf("expected openSQLite error")
	}
}

func TestImportFromSource_WithOnlyTasksTable(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "onlytasks.sqlite")
	target := filepath.Join(tmp, "target.sqlite")

	sdb, err := sql.Open("sqlite3", source)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = sdb.Exec(`CREATE TABLE tasks (id TEXT,title TEXT,description TEXT,status TEXT,priority INTEGER,issue_type TEXT,assignee TEXT,owner TEXT,labels_json TEXT,deps_json TEXT,metadata_json TEXT,close_reason TEXT,created_at TEXT,updated_at TEXT,closed_at TEXT)`)
	_, _ = sdb.Exec(`INSERT INTO tasks(id,title,description,status,priority,issue_type,assignee,owner,labels_json,deps_json,metadata_json,close_reason,created_at,updated_at,closed_at) VALUES('bd-only','only','','open',1,'task','','','[]','[]','{}','','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z','')`)
	_ = sdb.Close()

	src, err := openSQLite(source)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	dst, err := store.Open(target)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	_ = store.Init(dst)

	report, err := importFromSource(src, dst, source, importOptions{})
	if err != nil {
		t.Fatalf("import with only tasks should succeed: %v", err)
	}
	if report.Tasks["created"] != 1 {
		t.Fatalf("expected one task created, report=%+v", report.Tasks)
	}
	if report.Deps["created"] != 0 || report.Comments["created"] != 0 {
		t.Fatalf("expected no deps/comments import, deps=%+v comments=%+v", report.Deps, report.Comments)
	}
}

func TestImportFromSourceBadRowsSkipAndDryRun(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "source.sqlite")
	target := filepath.Join(tmp, "target.sqlite")

	sdb, err := sql.Open("sqlite3", source)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = sdb.Exec(`CREATE TABLE tasks (id TEXT,title TEXT,description TEXT,status TEXT,priority INTEGER,issue_type TEXT,assignee TEXT,owner TEXT,labels_json TEXT,deps_json TEXT,metadata_json TEXT,close_reason TEXT,created_at TEXT,updated_at TEXT,closed_at TEXT)`)
	_, _ = sdb.Exec(`INSERT INTO tasks(id,title,status,priority,issue_type,created_at,updated_at) VALUES('', '', 'open', 1, 'task', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`)
	_ = sdb.Close()

	src, err := openSQLite(source)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	dst, err := store.Open(target)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	_ = store.Init(dst)

	report, err := importFromSource(src, dst, source, importOptions{DryRun: true})
	if err != nil {
		t.Fatalf("dry run import failed: %v", err)
	}
	if len(report.Warnings) == 0 {
		t.Fatalf("expected warning for bad row")
	}
}

func captureOutput(t *testing.T, fn func() int) (int, string, string) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr
	defer func() { os.Stdout, os.Stderr = oldOut, oldErr }()

	code := fn()
	_ = wOut.Close()
	_ = wErr.Close()
	outB, _ := io.ReadAll(rOut)
	errB, _ := io.ReadAll(rErr)
	return code, string(outB), string(errB)
}

func TestCmdImportBeads_UsageAndDiscoveryErrors(t *testing.T) {
	if code, _, _ := captureOutput(t, func() int { return cmdImportBeads([]string{"--wat"}) }); code != 2 {
		t.Fatalf("expected usage error 2 got %d", code)
	}

	oldDB := os.Getenv("BEADS_DATABASE")
	oldDir := os.Getenv("BEADS_DIR")
	_ = os.Setenv("BEADS_DATABASE", "")
	_ = os.Setenv("BEADS_DIR", "")
	defer func() {
		_ = os.Setenv("BEADS_DATABASE", oldDB)
		_ = os.Setenv("BEADS_DIR", oldDir)
	}()
	code, _, errOut := captureOutput(t, func() int { return cmdImportBeads([]string{"--source", "/definitely/missing.sqlite"}) })
	if code != 2 || !strings.Contains(errOut, "no such file") {
		t.Fatalf("expected source error 2 with missing file, code=%d err=%q", code, errOut)
	}
}

func TestCmdImportBeads_NoCommentsJSON(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "target.sqlite")
	source := filepath.Join(tmp, "source.sqlite")
	seedSourceDB(t, source)

	oldSQ := os.Getenv("SQ_DB_PATH")
	_ = os.Setenv("SQ_DB_PATH", target)
	defer func() { _ = os.Setenv("SQ_DB_PATH", oldSQ) }()

	code, out, _ := captureOutput(t, func() int { return cmdImportBeads([]string{"--source", source, "--no-comments", "--json"}) })
	if code != 0 || !strings.Contains(out, "\"dry_run\": false") {
		t.Fatalf("expected json success output code=0 got code=%d out=%q", code, out)
	}
	if got := openCount(t, target, "comments"); got != 0 {
		t.Fatalf("expected no comments imported with --no-comments, got %d", got)
	}
}

func TestCmdImportBeads_Code4OnDependencyReadFailure(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "target.sqlite")
	source := filepath.Join(tmp, "baddeps.sqlite")

	db, err := sql.Open("sqlite3", source)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = db.Exec(`CREATE TABLE tasks (id TEXT,title TEXT,description TEXT,status TEXT,priority INTEGER,issue_type TEXT,assignee TEXT,owner TEXT,labels_json TEXT,deps_json TEXT,metadata_json TEXT,close_reason TEXT,created_at TEXT,updated_at TEXT,closed_at TEXT)`)
	_, _ = db.Exec(`INSERT INTO tasks(id,title,status,priority,issue_type,created_at,updated_at) VALUES('bd-1','ok','open',1,'task','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`)
	// malformed dependencies table (missing expected columns)
	_, _ = db.Exec(`CREATE TABLE dependencies (x TEXT)`)
	_ = db.Close()

	oldSQ := os.Getenv("SQ_DB_PATH")
	_ = os.Setenv("SQ_DB_PATH", target)
	defer func() { _ = os.Setenv("SQ_DB_PATH", oldSQ) }()

	code, _, errOut := captureOutput(t, func() int { return cmdImportBeads([]string{"--source", source, "--json"}) })
	if code != 4 || !strings.Contains(strings.ToLower(errOut), "dependencies") {
		t.Fatalf("expected code=4 dependency-read failure, got code=%d err=%q", code, errOut)
	}
}

func TestImportFromSource_IssuesSchemaVariant(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "issues.sqlite")
	target := filepath.Join(tmp, "target.sqlite")

	sdb, err := sql.Open("sqlite3", source)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = sdb.Exec(`CREATE TABLE issues (id TEXT,title TEXT,description TEXT,status TEXT,priority INTEGER,issue_type TEXT,assignee TEXT,owner TEXT,metadata_json TEXT,close_reason TEXT,created_at TEXT,updated_at TEXT,closed_at TEXT)`)
	_, _ = sdb.Exec(`CREATE TABLE issue_labels (issue_id TEXT,label TEXT)`)
	_, _ = sdb.Exec(`CREATE TABLE issue_deps (issue_id TEXT,depends_on_id TEXT,dep_type TEXT)`)
	_, _ = sdb.Exec(`INSERT INTO issues(id,title,description,status,priority,issue_type,assignee,owner,metadata_json,close_reason,created_at,updated_at,closed_at) VALUES('bd-a','A','desc','open',1,'task','alice','alice','{}','','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z','')`)
	_, _ = sdb.Exec(`INSERT INTO issues(id,title,description,status,priority,issue_type,assignee,owner,metadata_json,close_reason,created_at,updated_at,closed_at) VALUES('bd-b','B','desc','open',2,'task','','','{}','','2026-01-01T00:00:01Z','2026-01-01T00:00:01Z','')`)
	_, _ = sdb.Exec(`INSERT INTO issue_labels(issue_id,label) VALUES('bd-a','triage')`)
	_, _ = sdb.Exec(`INSERT INTO issue_deps(issue_id,depends_on_id,dep_type) VALUES('bd-a','bd-b','blocks')`)
	_ = sdb.Close()

	src, err := openSQLite(source)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	dst, err := store.Open(target)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	_ = store.Init(dst)

	report, err := importFromSource(src, dst, source, importOptions{})
	if err != nil {
		t.Fatalf("issues variant import failed: %v", err)
	}
	if report.Tasks["created"] != 2 {
		t.Fatalf("expected 2 created tasks, got %+v", report.Tasks)
	}
	if report.Deps["created"] != 1 {
		t.Fatalf("expected 1 dependency created, got %+v", report.Deps)
	}
}

func TestCmdImportBeads_SchemaValidationAcceptsIssuesTable(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "issues_only.sqlite")
	db, err := sql.Open("sqlite3", source)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = db.Exec(`CREATE TABLE issues (id TEXT,title TEXT)`)
	_ = db.Close()

	src, err := openSQLite(source)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	if err := validateSourceSchema(src); err != nil {
		t.Fatalf("expected issues schema to validate: %v", err)
	}
}

func TestHelpers_ToIntAndIssueAuxReaders(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "helpers.sqlite")
	db, err := sql.Open("sqlite3", source)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = db.Exec(`CREATE TABLE issues (id TEXT,title TEXT)`)
	_, _ = db.Exec(`INSERT INTO issues(id,title) VALUES('bd-1','one')`)
	_, _ = db.Exec(`CREATE TABLE issue_labels (issue_id TEXT,label TEXT)`)
	_, _ = db.Exec(`CREATE TABLE issue_deps (issue_id TEXT,depends_on_id TEXT,dep_type TEXT)`)
	_, _ = db.Exec(`INSERT INTO issue_labels(issue_id,label) VALUES('bd-1','a')`)
	_, _ = db.Exec(`INSERT INTO issue_labels(issue_id,label) VALUES('bd-1','')`)
	_, _ = db.Exec(`INSERT INTO issue_deps(issue_id,depends_on_id,dep_type) VALUES('bd-1','bd-2','blocks')`)
	_ = db.Close()

	src, err := openSQLite(source)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	labels, err := loadIssueLabelsJSON(src, "bd-1")
	if err != nil || !strings.Contains(labels, "a") {
		t.Fatalf("expected labels json with 'a', got=%q err=%v", labels, err)
	}
	deps, err := loadIssueDepsJSON(src, "bd-1")
	if err != nil || !strings.Contains(deps, "bd-2") {
		t.Fatalf("expected deps json with 'bd-2', got=%q err=%v", deps, err)
	}

	if got := toInt("7", 2); got != 7 {
		t.Fatalf("toInt parse expected 7 got %d", got)
	}
	if got := toInt("not-int", 2); got != 2 {
		t.Fatalf("toInt fallback expected 2 got %d", got)
	}
}

func TestImportFromSource_IssuesWithoutAuxTables(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "issues-no-aux.sqlite")
	target := filepath.Join(tmp, "target.sqlite")

	db, err := sql.Open("sqlite3", source)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = db.Exec(`CREATE TABLE issues (id TEXT,title TEXT,description TEXT,status TEXT,priority INTEGER,issue_type TEXT,assignee TEXT,owner TEXT,metadata_json TEXT,close_reason TEXT,created_at TEXT,updated_at TEXT,closed_at TEXT)`)
	_, _ = db.Exec(`INSERT INTO issues(id,title,description,status,priority,issue_type,assignee,owner,metadata_json,close_reason,created_at,updated_at,closed_at) VALUES('bd-z','Z','desc','open',2,'task','','','{}','','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z','')`)
	_ = db.Close()

	src, err := openSQLite(source)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	dst, err := store.Open(target)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	_ = store.Init(dst)

	report, err := importFromSource(src, dst, source, importOptions{})
	if err != nil {
		t.Fatalf("issues w/o aux import failed: %v", err)
	}
	if report.Tasks["created"] != 1 {
		t.Fatalf("expected one created task, got %+v", report.Tasks)
	}
}

func TestCmdImportBeads_SuccessPlainTextAndMappingError(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "target.sqlite")
	source := filepath.Join(tmp, "source.sqlite")
	seedSourceDB(t, source)

	oldSQ := os.Getenv("SQ_DB_PATH")
	_ = os.Setenv("SQ_DB_PATH", target)
	defer func() { _ = os.Setenv("SQ_DB_PATH", oldSQ) }()

	code, out, _ := captureOutput(t, func() int { return cmdImportBeads([]string{"--source", source}) })
	if code != 0 || !strings.Contains(out, "Imported from") {
		t.Fatalf("expected plain success output code=0 got code=%d out=%q", code, out)
	}

	bad := filepath.Join(tmp, "badmap.sqlite")
	db, err := sql.Open("sqlite3", bad)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = db.Exec(`CREATE TABLE tasks (id TEXT,title TEXT,description TEXT,status TEXT,priority TEXT,issue_type TEXT,assignee TEXT,owner TEXT,labels_json TEXT,deps_json TEXT,metadata_json TEXT,close_reason TEXT,created_at TEXT,updated_at TEXT,closed_at TEXT)`)
	_, _ = db.Exec(`INSERT INTO tasks(id,title,description,status,priority,issue_type,assignee,owner,labels_json,deps_json,metadata_json,close_reason,created_at,updated_at,closed_at) VALUES('bd-x','x','','open','not-int','task','','','[]','[]','{}','','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z','')`)
	_ = db.Close()

	code, _, errOut := captureOutput(t, func() int { return cmdImportBeads([]string{"--source", bad, "--json"}) })
	if code != 3 || !strings.Contains(strings.ToLower(errOut), "mapping") {
		t.Fatalf("expected mapping error code=3, got code=%d err=%q", code, errOut)
	}
}
