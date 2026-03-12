package cli

import (
	"testing"

	"github.com/berdon/squids/internal/store"
)

func TestReadyHelperBranchesLegacy(t *testing.T) {
	if got := splitCSV(" a, b ,,c "); len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("splitCSV unexpected: %#v", got)
	}
	if !containsString([]string{"x", "y"}, "y") || containsString([]string{"x", "y"}, "z") {
		t.Fatalf("containsString unexpected")
	}
	if !matchesAllLabels([]string{"backend", "urgent"}, []string{"backend"}) {
		t.Fatalf("matchesAllLabels expected true")
	}
	if matchesAllLabels([]string{"backend"}, []string{"backend", "urgent"}) {
		t.Fatalf("matchesAllLabels expected false")
	}
	if !matchesAnyLabel([]string{"backend", "urgent"}, []string{"frontend", "urgent"}) {
		t.Fatalf("matchesAnyLabel expected true")
	}
	if matchesAnyLabel([]string{"backend"}, []string{"frontend", "api"}) {
		t.Fatalf("matchesAnyLabel expected false")
	}
	if !matchesMetadataFields(map[string]string{"team": "platform", "env": "prod"}, []string{"team=platform", "env=prod"}) {
		t.Fatalf("matchesMetadataFields expected true")
	}
	if matchesMetadataFields(map[string]string{"team": "platform"}, []string{"team"}) {
		t.Fatalf("matchesMetadataFields expected false for malformed field")
	}
	if matchesMetadataFields(map[string]string{"team": "platform"}, []string{"team=infra"}) {
		t.Fatalf("matchesMetadataFields expected false for mismatch")
	}
}

func TestSortReadyTasksBranchesLegacy(t *testing.T) {
	base := []store.Task{
		{ID: "bd-b", Priority: 2, CreatedAt: "2026-03-12T02:00:00Z"},
		{ID: "bd-a", Priority: 1, CreatedAt: "2026-03-12T03:00:00Z"},
		{ID: "bd-c", Priority: 1, CreatedAt: "2026-03-12T01:00:00Z"},
	}

	prioritySorted := append([]store.Task(nil), base...)
	sortReadyTasks(prioritySorted, "priority")
	if prioritySorted[0].ID != "bd-c" || prioritySorted[1].ID != "bd-a" || prioritySorted[2].ID != "bd-b" {
		t.Fatalf("priority sort unexpected: %#v", prioritySorted)
	}

	hybridSorted := append([]store.Task(nil), base...)
	sortReadyTasks(hybridSorted, "hybrid")
	if hybridSorted[0].ID != "bd-c" || hybridSorted[1].ID != "bd-a" || hybridSorted[2].ID != "bd-b" {
		t.Fatalf("hybrid sort unexpected: %#v", hybridSorted)
	}

	oldestSorted := append([]store.Task(nil), base...)
	sortReadyTasks(oldestSorted, "oldest")
	if oldestSorted[0].ID != "bd-c" || oldestSorted[1].ID != "bd-b" || oldestSorted[2].ID != "bd-a" {
		t.Fatalf("oldest sort unexpected: %#v", oldestSorted)
	}

	unknownSorted := append([]store.Task(nil), base...)
	sortReadyTasks(unknownSorted, "wat")
	if unknownSorted[0].ID != "bd-b" || unknownSorted[1].ID != "bd-a" || unknownSorted[2].ID != "bd-c" {
		t.Fatalf("unknown sort should preserve order: %#v", unknownSorted)
	}
}
