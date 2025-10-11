package ast_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cmessinides/kiwi/internal/ast"
	"github.com/cmessinides/kiwi/internal/testfmt"
)

// Helper for constructing expected output.
func o(lines ...string) string {
	return strings.Join(lines, "\n")
}

var printerTests = []struct {
	label   string
	options *ast.PrinterOptions
	input   *ast.Node[string]
	output  string
}{
	{
		"single node",
		&ast.PrinterOptions{},
		ast.NewNode("document"),
		o("(document)"),
	},
	{
		"nested nodes",
		&ast.PrinterOptions{},
		ast.NewNode("document").
			Append(
				ast.NewNode("paragraph"),
				ast.NewNode("blockquote").
					Append(ast.NewNode("paragraph")),
				ast.NewNode("paragraph"),
			),
		o(
			"(document",
			"\t(paragraph)",
			"\t(blockquote",
			"\t\t(paragraph))",
			"\t(paragraph))",
		),
	},
	{
		"attributes",
		&ast.PrinterOptions{},
		ast.NewNode("document").
			SetAttr("lang", "en-US").
			Append(
				ast.NewNode("heading").
					SetAttr("level", 1).
					SetAttr("id", "title"),
			),
		o(
			"(document \"en-US\"",
			"\t(heading 1 \"title\"))",
		),
	},
	{
		"attribute-formatters",
		&ast.PrinterOptions{
			AttrStringers: ast.AttrStringerMap{
				"lang": func(value any) fmt.Stringer {
					if s, ok := value.(string); ok {
						value = strings.ToLower(s)
					}

					return ast.DefaultAttrStringer{value}
				},
			},
		},
		ast.NewNode("document").
			SetAttr("title", "should not change").
			SetAttr("lang", "en-US"),
		o(
			"(document \"should not change\" \"en-us\")",
		),
	},
	{
		"include-keys",
		&ast.PrinterOptions{IncludeAttrKeys: true},
		ast.NewNode("document").
			SetAttr("lang", "en-US").
			Append(
				ast.NewNode("heading").
					SetAttr("level", 1).
					SetAttr("id", "title"),
			),
		o(
			"(document lang=\"en-US\"",
			"\t(heading level=1 id=\"title\"))",
		),
	},
	{
		"custom-indent",
		&ast.PrinterOptions{IndentChars: ">   "},
		ast.NewNode("document").
			Append(
				ast.NewNode("paragraph"),
				ast.NewNode("blockquote").
					Append(ast.NewNode("paragraph")),
			),
		o(
			"(document",
			">   (paragraph)",
			">   (blockquote",
			">   >   (paragraph)))",
		),
	},
}

func TestPrinter(t *testing.T) {
	for _, tt := range printerTests {
		s := new(strings.Builder)
		p := ast.NewPrinter[string](s, tt.options)

		err := p.Print(tt.input)
		if err != nil {
			t.Errorf("%s: unexpected error: %s", tt.label, err)
			continue
		}

		expected := tt.output
		actual := s.String()

		if expected != actual {
			t.Errorf(
				"%s: printer output did not match expected\n\n%s",
				tt.label,
				testfmt.Compare(expected, actual),
			)
		}
	}
}
