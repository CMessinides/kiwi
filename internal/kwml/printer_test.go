package kwml_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cmessinides/kiwi/internal/kwml"
	"github.com/cmessinides/kiwi/internal/testfmt"
)

func TestPrinter(t *testing.T) {
	doc := func(blocks ...kwml.Node) kwml.Node {
		return &kwml.Document{
			Blocks: blocks,
		}
	}

	h := func(level int, spans ...kwml.Node) kwml.Node {
		return &kwml.Heading{
			Level: level,
			Spans: spans,
		}
	}

	p := func(spans ...kwml.Node) kwml.Node {
		return &kwml.Paragraph{Spans: spans}
	}

	txt := func(content string) kwml.Node {
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
			o("(document)"),
		},
		{
			"paragraphs",
			doc(
				p(txt("Hello world")),
				p(txt("A multi-line\nparagraph")),
			),
			o(
				"(document",
				"\t(paragraph",
				"\t\t(text content=\"Hello world\"))",
				"\t(paragraph",
				"\t\t(text content=\"A multi-line\\nparagraph\")))",
			),
		},
		{
			"headings",
			doc(
				h(1),
				h(2, txt("Section heading")),
			),
			o(
				"(document",
				"\t(heading level=1)",
				"\t(heading level=2",
				"\t\t(text content=\"Section heading\")))",
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
