package cli

import (
	"testing"

	"github.com/berdon/squids/internal/store"
)

func TestContainsString(t *testing.T) {
	if !containsString([]string{"a", "b"}, "b") {
		t.Fatalf("expected to find string")
	}
	if containsString([]string{"a", "b"}, "c") {
		t.Fatalf("did not expect to find string")
	}
}

func TestMatchesAnyLabel(t *testing.T) {
	if !matchesAnyLabel([]string{"bug", "p1"}, nil) {
		t.Fatalf("empty allowed should match")
	}
	if !matchesAnyLabel([]string{"bug", "p1"}, []string{"p1", "p2"}) {
		t.Fatalf("expected overlapping label to match")
	}
	if matchesAnyLabel([]string{"bug", "p1"}, []string{"p2", "p3"}) {
		t.Fatalf("expected no overlap to fail")
	}
}

func TestMatchesMetadataFields(t *testing.T) {
	meta := map[string]string{"env": "prod", "team": "cli"}
	if !matchesMetadataFields(meta, []string{"env=prod", "team=cli"}) {
		t.Fatalf("expected metadata fields to match")
	}
	if matchesMetadataFields(meta, []string{"env=staging"}) {
		t.Fatalf("expected mismatched metadata to fail")
	}
	if matchesMetadataFields(meta, []string{"bad-format"}) {
		t.Fatalf("expected invalid metadata field format to fail")
	}
}

func TestSortReadyTasks(t *testing.T) {
	tasks := []store.Task{
		{ID: "bd-2", Priority: 2, CreatedAt: "2026-03-02T00:00:00Z"},
		{ID: "bd-1", Priority: 1, CreatedAt: "2026-03-03T00:00:00Z"},
		{ID: "bd-3", Priority: 1, CreatedAt: "2026-03-01T00:00:00Z"},
	}
	sortReadyTasks(tasks, "priority")
	if tasks[0].ID != "bd-3" || tasks[1].ID != "bd-1" || tasks[2].ID != "bd-2" {
		t.Fatalf("unexpected priority sort order: %+v", tasks)
	}

	tasks = []store.Task{
		{ID: "bd-2", Priority: 2, CreatedAt: "2026-03-02T00:00:00Z"},
		{ID: "bd-1", Priority: 1, CreatedAt: "2026-03-03T00:00:00Z"},
		{ID: "bd-3", Priority: 1, CreatedAt: "2026-03-01T00:00:00Z"},
	}
	sortReadyTasks(tasks, "oldest")
	if tasks[0].ID != "bd-3" || tasks[1].ID != "bd-2" || tasks[2].ID != "bd-1" {
		t.Fatalf("unexpected oldest sort order: %+v", tasks)
	}

	tasks = []store.Task{{ID: "bd-1"}, {ID: "bd-2"}}
	sortReadyTasks(tasks, "unknown")
	if tasks[0].ID != "bd-1" || tasks[1].ID != "bd-2" {
		t.Fatalf("unknown sort should preserve order: %+v", tasks)
	}
}
