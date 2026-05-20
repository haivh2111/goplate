package main

import (
	"os"

	"github.com/haivh2111/goplate/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
