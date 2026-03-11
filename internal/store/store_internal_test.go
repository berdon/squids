package store

import (
	"testing"
)

func TestCollisionProbabilityMonotonic(t *testing.T) {
	p1 := collisionProbability(10, 4)
	p2 := collisionProbability(100, 4)
	if !(p2 > p1) {
		t.Fatalf("expected probability to increase with more issues: p1=%f p2=%f", p1, p2)
	}
}

func TestComputeAdaptiveLengthBounds(t *testing.T) {
	got := computeAdaptiveLength(1000, 3, 8, 0.000001)
	if got < 3 || got > 8 {
		t.Fatalf("out of bounds length: %d", got)
	}

	// If no length satisfies, should return max.
	got = computeAdaptiveLength(1_000_000, 3, 5, 0.0)
	if got != 5 {
		t.Fatalf("expected max length fallback=5 got %d", got)
	}
}
