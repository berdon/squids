package main

import (
	"os"

	"github.com/berdon/squids/internal/cli"
)

func run(args []string) int {
	return cli.Run(args)
}

var exitFn = os.Exit

func main() {
	exitFn(run(os.Args[1:]))
}
