package main

import (
	"os"

	"gitea/auhanson/squids/internal/cli"
)

func run(args []string) int {
	return cli.Run(args)
}

func main() {
	os.Exit(run(os.Args[1:]))
}
