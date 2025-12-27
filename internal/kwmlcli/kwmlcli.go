// Package kwmlcli provides a CLI for parsing, validating, and converting KWML
// documents to HTML.
package kwmlcli

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/cmessinides/kiwi/internal/kwml"
)

var cli struct {
	Paths []string `arg:"" optional:"" help:"Files to parse" type:"existingfile"`
}

// Run executes the CLI.
func Run() {
	var failed bool

	ctx := kong.Parse(&cli)
	if len(cli.Paths) == 0 {
		ctx.PrintUsage(false)
		ctx.Fatalf("at least one path is required")
	}

	multiple := len(cli.Paths) > 1
	files := make([]*os.File, len(cli.Paths))
	defer func() {
		for _, f := range files {
			if f != nil {
				f.Close()
			}
		}
	}()

	for i, p := range cli.Paths {
		if p == "-" {
			files[i] = os.Stdin
		} else {
			f, err := os.Open(p)
			if err != nil {
				failed = true
				ctx.Errorf("could not open %s: %s", p, err)
			} else {
				files[i] = f
			}

		}
	}

	for i, f := range files {
		if f == nil {
			continue
		}

		fname := f.Name()
		if f == os.Stdin {
			fname = "stdin"
		}

		if multiple {
			fmt.Fprintf(ctx.Stdout, "%s:\n", fname)
		}

		doc, err := kwml.Parse(f)
		if err != nil {
			failed = true
			ctx.Errorf("failed to parse %s: %s", fname, err)
		}

		err = kwml.PrintAST(ctx.Stdout, doc)
		if err != nil {
			failed = true
			ctx.Errorf("failed to print AST for %s: %s", fname, err)
		}

		isLast := i == len(files)-1
		if multiple && !isLast {
			fmt.Fprint(ctx.Stdout, "\n")
		}
	}

	if failed {
		ctx.Fatalf("encountered a parsing error")
	}
}
