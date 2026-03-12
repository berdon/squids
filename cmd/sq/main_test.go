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

func TestRunUnknownCommandReturnsNonZero(t *testing.T) {
	if code := run([]string{"definitely-unknown-command"}); code == 0 {
		t.Fatalf("expected unknown command non-zero exit")
	}
}

func TestMainInvokesExitFnWithRunCode(t *testing.T) {
	oldArgs := os.Args
	oldExit := exitFn
	defer func() {
		os.Args = oldArgs
		exitFn = oldExit
	}()

	os.Args = []string{"sq", "help"}
	got := -1
	exitFn = func(code int) {
		got = code
	}

	main()
	if got != 0 {
		t.Fatalf("expected main to call exitFn(0), got %d", got)
	}
}

func TestMainInvokesExitFnWithNonZeroCode(t *testing.T) {
	oldArgs := os.Args
	oldExit := exitFn
	defer func() {
		os.Args = oldArgs
		exitFn = oldExit
	}()

	os.Args = []string{"sq", "definitely-unknown-command"}
	got := -1
	exitFn = func(code int) {
		got = code
	}

	main()
	if got == 0 {
		t.Fatalf("expected main to call non-zero exit code for unknown command")
	}
}
