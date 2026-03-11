package main

import (
	"os"

	"gitea/auhanson/squids/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
