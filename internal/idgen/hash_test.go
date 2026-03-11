package idgen

import (
	"strings"
	"testing"
	"time"
)

func TestEncodeBase36_LengthAndPadding(t *testing.T) {
	got := EncodeBase36([]byte{0x01}, 4)
	if len(got) != 4 {
		t.Fatalf("expected len 4 got %d", len(got))
	}
	if got == "" {
		t.Fatalf("empty output")
	}
}

func TestGenerateHashID_DeterministicAndPrefixed(t *testing.T) {
	ts := time.Unix(1700000000, 123)
	a := GenerateHashID("bd", "title", "desc", "me", ts, 3, 0)
	b := GenerateHashID("bd", "title", "desc", "me", ts, 3, 0)
	if a != b {
		t.Fatalf("expected deterministic ids: %q != %q", a, b)
	}
	if !strings.HasPrefix(a, "bd-") {
		t.Fatalf("expected prefix bd- got %q", a)
	}
}

func TestGenerateHashID_LengthVariants(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	for _, n := range []int{3, 4, 5, 6, 7, 8, 99} {
		id := GenerateHashID("sq", "t", "d", "c", ts, n, 1)
		parts := strings.Split(id, "-")
		if len(parts) != 2 {
			t.Fatalf("bad id format: %q", id)
		}
		if n >= 3 && n <= 8 {
			if len(parts[1]) != n {
				t.Fatalf("expected hash len %d got %d for %q", n, len(parts[1]), id)
			}
		}
	}
}
