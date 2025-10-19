package kwml

import (
	"fmt"
	"io"
	"strings"
)

type Printer struct {
	w        io.Writer
	err      error
	depth    int
	dangling int
	started  bool
}

func (p *Printer) StartVisit(node Node) (v Visitor) {
	if p.err != nil {
		return nil
	}

	if p.started {
		p.closeDangling()
		if p.err != nil {
			return nil
		}
	} else {
		p.started = true
	}

	p.dangling = 0
	p.depth++

	switch v := node.(type) {
	case *Document:
		p.printNode("document")
	case *Heading:
		p.printNode("heading", "level", v.Level)
	case *Paragraph:
		p.printNode("paragraph")
	case *Blockquote:
		p.printNode("blockquote")
	case *OrderedList:
		p.printNode("ordered_list")
	case *UnorderedList:
		p.printNode("unordered_list")
	case *ListItem:
		p.printNode("list_item")
	case *Verbatim:
		p.printNode(
			"verbatim",
			"lang", v.Lang,
			"content", string(v.Content),
		)
	case *Macro:
		p.printNode(
			"macro",
			"tag", v.Tag,
			"content", string(v.Content),
		)
	case *Text:
		p.printNode("text", "content", string(v.Content))
	case *Emphasis:
		p.printNode("emphasis")
	case *StrongEmphasis:
		p.printNode("strong_emphasis")
	case *Link:
		p.printNode("link", "target", v.Target)
	case *WikiLink:
		p.printNode("wiki_link", "target", v.Target)
	}

	if p.err != nil {
		return nil
	}

	return p
}

func (p *Printer) EndVisit(node Node) {
	p.depth--
	p.dangling++

	if p.depth == 0 {
		p.closeDangling()
	}
}

func (p *Printer) Err() error {
	return p.err
}

func (p *Printer) printNode(name string, attrs ...any) error {
	indent := strings.Repeat("\t", p.depth-1)
	if _, p.err = fmt.Fprint(p.w, indent, "(", name); p.err != nil {
		return p.err
	}

	for i := 0; i < len(attrs); i++ {
		key := attrs[i]
		var v any
		if i+1 < len(attrs) {
			i++
			v = attrs[i]
		}

		if _, p.err = fmt.Fprintf(p.w, " %s=%#v", key, v); p.err != nil {
			return p.err
		}
	}

	return nil
}

func (p *Printer) closeDangling() {
	// Close any dangling nodes and start a new line.
	parens := strings.Repeat(")", p.dangling)
	_, p.err = fmt.Fprintln(p.w, parens)
}

func PrintAST(w io.Writer, root Node) error {
	p := &Printer{w: w}
	Walk(root, p)
	return p.Err()
}
