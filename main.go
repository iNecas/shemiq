package main

import (
	"os"

	"github.com/iNecas/shemiq/cmd"
)

func main() {
	if err := cmd.Execute(os.Stdin, os.Stdout, os.Stderr, os.Args[1:]); err != nil {
		os.Exit(1) // Cobra has already printed the error and usage to stderr.
	}
}
