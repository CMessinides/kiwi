package kwml

import (
	"fmt"
	"io"
	"strings"
)

type Printer struct {
	w     io.Writer
	err   error
	depth int
}

func (p *Printer) Print(node Node) error {
	p.printNode(node)
	if p.err != nil {
		return p.err
	}

	_, p.err = fmt.Fprint(p.w, "\n")
	return p.err
}

func (p *Printer) Err() error {
	return p.err
}

func (p *Printer) printNode(node Node) error {
	if p.err != nil {
		return p.err
	}

	if _, p.err = fmt.Fprint(p.w, "("); p.err != nil {
		return p.err
	}

	switch v := node.(type) {
	case *Document:
		p.printHeader("document")
	case *Heading:
		p.printHeader("heading", "level", v.Level)
	case *Paragraph:
		p.printHeader("paragraph")
	case *Blockquote:
		p.printHeader("blockquote")
	case *OrderedList:
		p.printHeader("ordered_list")
	case *UnorderedList:
		p.printHeader("unordered_list")
	case *ListItem:
		p.printHeader("list_item")
	case *Verbatim:
		p.printHeader(
			"verbatim",
			"lang", v.Lang,
			"content", string(v.Raw),
		)
	case *Macro:
		p.printHeader(
			"macro",
			"tag", v.Tag,
			"content", string(v.Raw),
		)
	case *Text:
		p.printHeader("text", "content", string(v.Content))
	case *Emphasis:
		p.printHeader("emphasis")
	case *StrongEmphasis:
		p.printHeader("strong_emphasis")
	case *Link:
		p.printHeader("link", "target", v.Target)
	case *WikiLink:
		p.printHeader("wiki_link", "target", v.Target)
	default:
		panic(fmt.Sprintf("unexpected node type: %T", v))
	}

	if p.err != nil {
		return p.err
	}

	if i, ok := node.(childrenIterator); ok {
		p.depth++
		indent := strings.Repeat("\t", p.depth)
		for c := range i.iterChildren() {
			if _, p.err = fmt.Fprint(p.w, "\n", indent); p.err != nil {
				return p.err
			}

			p.printNode(c)
			if p.err != nil {
				return p.err
			}
		}
		p.depth--
	}

	_, p.err = fmt.Fprint(p.w, ")")
	return p.err
}

func (p *Printer) printHeader(name string, attrs ...any) error {
	if _, p.err = fmt.Fprint(p.w, name); p.err != nil {
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

func PrintAST(w io.Writer, root Node) error {
	p := &Printer{w: w}
	return p.Print(root)
}
