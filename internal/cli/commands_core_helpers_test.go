package cli

import (
	"testing"

	"github.com/berdon/squids/internal/store"
)

func TestCommandCoreHelpers(t *testing.T) {
	if got := splitCSV(" a, ,b , c "); len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("unexpected splitCSV result: %#v", got)
	}
	if !containsString([]string{"a", "b"}, "b") {
		t.Fatalf("expected containsString hit")
	}
	if containsString([]string{"a", "b"}, "c") {
		t.Fatalf("expected containsString miss")
	}
	if !matchesAllLabels([]string{"bug", "urgent", "backend"}, []string{"bug", "backend"}) {
		t.Fatalf("expected matchesAllLabels hit")
	}
	if matchesAllLabels([]string{"bug"}, []string{"bug", "backend"}) {
		t.Fatalf("expected matchesAllLabels miss")
	}
	if !matchesAnyLabel([]string{"bug", "urgent"}, []string{"docs", "urgent"}) {
		t.Fatalf("expected matchesAnyLabel hit")
	}
	if matchesAnyLabel([]string{"bug"}, []string{"docs", "feature"}) {
		t.Fatalf("expected matchesAnyLabel miss")
	}
	if !matchesAnyLabel([]string{"bug"}, nil) {
		t.Fatalf("expected empty allowed labels to match")
	}
	if !matchesMetadataFields(map[string]string{"team": "platform", "env": "prod"}, []string{"team=platform", "env=prod"}) {
		t.Fatalf("expected metadata match")
	}
	if matchesMetadataFields(map[string]string{"team": "platform"}, []string{"team"}) {
		t.Fatalf("expected malformed metadata filter to fail")
	}
	if matchesMetadataFields(map[string]string{"team": "platform"}, []string{"team=infra"}) {
		t.Fatalf("expected metadata mismatch to fail")
	}
}

func TestSortReadyTasksBranches(t *testing.T) {
	base := []store.Task{
		{ID: "bd-2", Priority: 2, CreatedAt: "2026-01-02T00:00:00Z"},
		{ID: "bd-1", Priority: 1, CreatedAt: "2026-01-03T00:00:00Z"},
		{ID: "bd-3", Priority: 1, CreatedAt: "2026-01-01T00:00:00Z"},
	}

	prioritySorted := append([]store.Task(nil), base...)
	sortReadyTasks(prioritySorted, "priority")
	if prioritySorted[0].ID != "bd-3" || prioritySorted[1].ID != "bd-1" || prioritySorted[2].ID != "bd-2" {
		t.Fatalf("unexpected priority sort order: %#v", prioritySorted)
	}

	hybridSorted := append([]store.Task(nil), base...)
	sortReadyTasks(hybridSorted, "hybrid")
	if hybridSorted[0].ID != "bd-3" {
		t.Fatalf("expected hybrid sort to follow priority branch: %#v", hybridSorted)
	}

	oldestSorted := append([]store.Task(nil), base...)
	sortReadyTasks(oldestSorted, "oldest")
	if oldestSorted[0].ID != "bd-3" || oldestSorted[2].ID != "bd-1" {
		t.Fatalf("unexpected oldest sort order: %#v", oldestSorted)
	}

	unknownSorted := append([]store.Task(nil), base...)
	sortReadyTasks(unknownSorted, "wat")
	if unknownSorted[0].ID != base[0].ID || unknownSorted[1].ID != base[1].ID || unknownSorted[2].ID != base[2].ID {
		t.Fatalf("expected unknown sort to preserve order: %#v", unknownSorted)
	}
}
