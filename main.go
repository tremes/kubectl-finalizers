package main

import (
	"os"

	"github.com/tremes/kubectl-finalizers/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
