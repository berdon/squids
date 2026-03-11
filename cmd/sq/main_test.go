package main

import "testing"

func TestRunHelp(t *testing.T) {
	if code := run([]string{"help"}); code != 0 {
		t.Fatalf("expected help code 0 got %d", code)
	}
}
