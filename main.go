package main

import (
	"os"

	"orangebuilder/src/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:]))
}
