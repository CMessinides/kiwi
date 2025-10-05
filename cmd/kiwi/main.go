package main

import (
	"os"

	"github.com/cmessinides/kiwi/internal/kiwicli"
)

func main() {
	os.Exit(kiwicli.Run(os.Args))
}
