package idgen

import (
	"strings"
	"testing"
	"time"
)

func TestEncodeBase36_ZeroAndTrimBranches(t *testing.T) {
	if got := EncodeBase36([]byte{0x00}, 4); got != "0000" {
		t.Fatalf("expected zero padding, got %q", got)
	}
	if got := EncodeBase36([]byte{0xff, 0xff, 0xff, 0xff}, 3); len(got) != 3 {
		t.Fatalf("expected trimmed length 3, got %q", got)
	}
}

func TestGenerateHashID_UsesRequestedPrefixAndNonce(t *testing.T) {
	ts := time.Unix(1700000000, 456)
	one := GenerateHashID("sq", "title", "desc", "me", ts, 6, 1)
	two := GenerateHashID("sq", "title", "desc", "me", ts, 6, 2)
	if !strings.HasPrefix(one, "sq-") {
		t.Fatalf("expected sq prefix, got %q", one)
	}
	if one == two {
		t.Fatalf("expected nonce to influence id, got %q and %q", one, two)
	}
}
