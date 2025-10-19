package kwml_test

import (
	"bytes"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/cmessinides/kiwi/internal/kwml"
	"github.com/cmessinides/kiwi/internal/testfmt"
)

func TestParser(t *testing.T) {
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

		output := name + ".ast"
		expected, err := os.ReadFile("testdata/examples/" + output)
		if err != nil {
			t.Errorf("%s: could not read %s: %s", name, output, err)
			continue
		}

		doc, err := kwml.Parse(srcFile)
		if err != nil {
			t.Errorf("%s: parse failed:\n\n%s", name, err)
			continue
		}

		buf := new(bytes.Buffer)
		if err = kwml.PrintAST(buf, doc); err != nil {
			t.Errorf("%s: could not print AST: %e", name, err)
			continue
		}

		actual := buf.Bytes()
		if !bytes.Equal(expected, actual) {
			t.Errorf(
				"%s: AST does not match expected:\n\n%s",
				name,
				testfmt.Compare(string(expected), string(actual)),
			)
		}
	}
}
