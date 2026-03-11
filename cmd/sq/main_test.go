package main

import (
	"os"
	"testing"
)

func TestRunHelp(t *testing.T) {
	if code := run([]string{"help"}); code != 0 {
		t.Fatalf("expected help code 0 got %d", code)
	}
}

func TestMainSetsExitCodeViaRunPath(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"sq", "help"}
	if code := run(os.Args[1:]); code != 0 {
		t.Fatalf("expected code 0 got %d", code)
	}
}
