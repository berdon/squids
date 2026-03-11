package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type result struct {
	Command string `json:"command"`
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

func printJSON(v any) int {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode json: %v\n", err)
		return 1
	}
	return 0
}

func usage() {
	fmt.Println("sq - squids task CLI")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  sq <command> [args]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  init    Initialize task storage")
	fmt.Println("  ready   Check backend readiness")
	fmt.Println("  create  Create a task")
	fmt.Println("  show    Show a task")
	fmt.Println("  list    List tasks")
	fmt.Println("  update  Update a task")
	fmt.Println("  close   Close a task")
}

func notImplemented(cmd string) int {
	return printJSON(result{Command: cmd, OK: false, Message: "not implemented yet"})
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "-h", "--help", "help":
		usage()
		os.Exit(0)
	case "init":
		os.Exit(notImplemented("init"))
	case "ready":
		os.Exit(notImplemented("ready"))
	case "create":
		os.Exit(notImplemented("create"))
	case "show":
		os.Exit(notImplemented("show"))
	case "list":
		os.Exit(notImplemented("list"))
	case "update":
		os.Exit(notImplemented("update"))
	case "close":
		os.Exit(notImplemented("close"))
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}
