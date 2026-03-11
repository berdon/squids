package cli

import (
	"fmt"
	"os"
)

// Run executes the sq CLI with pre-sliced args (excluding argv[0])
// and returns a process exit code.
func Run(args []string) int {
	if len(args) < 1 {
		usage()
		return 2
	}

	switch args[0] {
	case "-h", "--help", "help":
		usage()
		return 0
	case "init":
		return cmdInit()
	case "ready":
		return cmdReady()
	case "create":
		return cmdCreate(args[1:])
	case "q":
		return cmdQ(args[1:])
	case "show":
		return cmdShow(args[1:])
	case "list":
		return cmdList(args[1:])
	case "update":
		return cmdUpdate(args[1:])
	case "close":
		return cmdClose(args[1:])
	case "reopen":
		return cmdReopen(args[1:])
	case "delete":
		return cmdDelete(args[1:])
	case "label":
		return cmdLabel(args[1:])
	case "dep":
		return cmdDep(args[1:])
	case "comments":
		return cmdComments(args[1:])
	case "todo":
		return cmdTodo(args[1:])
	case "children":
		return cmdChildren(args[1:])
	case "blocked":
		return cmdBlocked(args[1:])
	case "defer":
		return cmdDefer(args[1:])
	case "undefer":
		return cmdUndefer(args[1:])
	case "duplicate":
		return cmdDuplicate(args[1:])
	case "supersede":
		return cmdSupersede(args[1:])
	case "types":
		return cmdTypes(args[1:])
	case "query":
		return cmdQuery(args[1:])
	case "search":
		return cmdSearch(args[1:])
	case "count":
		return cmdCount(args[1:])
	case "status", "stats":
		return cmdStatus()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		usage()
		return 2
	}
}
