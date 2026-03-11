package main

import (
	"os"

	"github.com/berdon/squids/internal/cli"
)

func run(args []string) int {
	return cli.Run(args)
}

func main() {
	os.Exit(run(os.Args[1:]))
}
