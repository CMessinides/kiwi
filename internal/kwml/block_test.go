package kwml_test

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/cmessinides/kiwi/internal/ast"
	"github.com/cmessinides/kiwi/internal/kwml"
	"github.com/cmessinides/kiwi/internal/testfmt"
)

func TestBlockParser(t *testing.T) {
	fixtures := os.DirFS("testdata/examples")
	inputs, err := fs.Glob(fixtures, "*.kwml")
	if err != nil {
		t.Fatalf("unexpected error reading fixture inputs: %s", err)
	}

	for _, input := range inputs {
		name := strings.TrimSuffix(input, ".kwml")

		srcFile, err := os.Open("testdata/examples/" + input)
		if err != nil {
			t.Errorf("%s: unexpected error reading %s: %s", name, input, err)
			continue
		}
		defer srcFile.Close()

		output := name + ".blocks.ast"
		outBytes, err := os.ReadFile("testdata/examples/" + output)
		if err != nil {
			t.Errorf("%s: could not read %s: %s", name, output, err)
			continue
		}

		b := kwml.NewBlockParser(srcFile)
		doc, err := b.Parse()
		if err != nil {
			t.Errorf("%s: parse failed:\n\n%s", name, err)
			continue
		}

		result := new(strings.Builder)
		printer := ast.NewPrinter[kwml.Block](result, &ast.PrinterOptions{
			AttrStringers: ast.AttrStringerMap{
				"content": func(value any) fmt.Stringer {
					if b, ok := value.([]byte); ok {
						value = string(b)
					}

					return ast.DefaultAttrStringer{Value: value}
				},
			},
			IncludeAttrKeys: true,
		})
		err = printer.Print(doc)
		if err != nil {
			t.Errorf("%s: could not print AST: %e", name, err)
			continue
		}

		expected := strings.TrimSuffix(string(outBytes), "\n")
		actual := result.String()
		if expected != actual {
			t.Errorf(
				"%s: AST does not match expected:\n\n%s",
				name,
				testfmt.Compare(expected, actual),
			)
		}
	}
}
