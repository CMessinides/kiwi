package kwml_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cmessinides/kiwi/internal/kwml"
	"github.com/cmessinides/kiwi/internal/testfmt"
)

func TestPrinter(t *testing.T) {
	doc := func(blocks ...kwml.BlockNode) *kwml.Document {
		return &kwml.Document{
			Children: blocks,
		}
	}

	h := func(level int, children ...kwml.InlineNode) *kwml.Heading {
		return &kwml.Heading{
			Level:    level,
			Children: children,
		}
	}

	p := func(children ...kwml.InlineNode) *kwml.Paragraph {
		return &kwml.Paragraph{Children: children}
	}

	txt := func(content string) *kwml.Text {
		return &kwml.Text{Content: []byte(content)}
	}

	// [O]utput -- helper for joining lines of text.
	o := func(lines ...string) string {
		return strings.Join(lines, "\n") + "\n"
	}

	printerCases := []struct {
		label    string
		input    kwml.Node
		expected string
	}{
		{
			"empty document",
			doc(),
			o("document"),
		},
		{
			"paragraphs",
			doc(
				p(txt("Hello world")),
				p(txt("A multi-line\nparagraph")),
			),
			o(
				"document",
				"\tparagraph",
				"\t\ttext content=\"Hello world\"",
				"\tparagraph",
				"\t\ttext content=\"A multi-line\\nparagraph\"",
			),
		},
		{
			"headings",
			doc(
				h(1),
				h(2, txt("Section heading")),
			),
			o(
				"document",
				"\theading level=1",
				"\theading level=2",
				"\t\ttext content=\"Section heading\"",
			),
		},
	}

	for _, tt := range printerCases {
		buf := new(bytes.Buffer)
		kwml.PrintAST(buf, tt.input)

		actual := buf.String()

		if actual != tt.expected {
			t.Errorf(
				"%s: printer output did not match expected:\n\n%s",
				tt.label,
				testfmt.Compare(tt.expected, actual),
			)
		}
	}
}
