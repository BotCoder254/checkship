package main

import (
	"os"

	"github.com/checkship-deploy/checkship/cmd"
)

func main() {
	cmd := cmd.Root()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
