package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] != "list" {
		fmt.Fprintln(os.Stderr, "usage: launchy list")
		os.Exit(2)
	}

	if err := runList(); err != nil {
		fmt.Fprintln(os.Stderr, "launchy:", err)
		os.Exit(1)
	}
}
