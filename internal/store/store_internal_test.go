package store

import "testing"

func TestInitAndCounterOnClosedDB(t *testing.T) {
	w, done := openTestDB(t)
	done()

	if err := Init(w.DB); err == nil {
		t.Fatalf("expected init on closed db to fail")
	}
	if _, err := nextCounterID(w.DB, "bd"); err == nil {
		t.Fatalf("expected nextCounterID on closed db to fail")
	}
}
