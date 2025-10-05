// Package kiwicli provides a CLI for managing and serving kiwi wikis.
package kiwicli

import "fmt"

// Run executes the CLI with the given arguments. It returns an OS exit code.
func Run(args []string) int {
	fmt.Println("Hello from kiwi")
	fmt.Printf("args: %v\n", args)
	return 0
}
