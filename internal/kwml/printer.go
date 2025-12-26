package kwml

import (
	"fmt"
	"io"
	"strings"
)

type printer struct {
	w     io.Writer
	err   error
	depth int
}

func (p *printer) StartVisit(node Node) (v Visitor) {
	if p.err != nil {
		return nil
	}

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
			"content", string(v.Raw),
		)
	case *Macro:
		p.printNode(
			"macro",
			"tag", v.Tag,
			"content", string(v.Raw),
		)
	case *Text:
		p.printNode("text", "content", string(v.Content))
	case *Emphasis:
		p.printNode("emphasis")
	case *StrongEmphasis:
		p.printNode("strong_emphasis")
	case *Code:
		p.printNode("code", "raw", string(v.Raw))
	case *Image:
		p.printNode("image", "target", v.Target)
	case *Link:
		p.printNode("link", "target", v.Target)
	case *WikiLink:
		p.printNode("wiki_link", "target", v.Target)
	default:
		panic(fmt.Sprintf("unexpected node type: %T", v))
	}

	if p.err == nil {
		_, p.err = fmt.Fprintln(p.w)
	}

	p.depth++
	return p
}

func (p *printer) EndVisit(node Node) {
	p.depth--
}

func (p *printer) Err() error {
	return p.err
}

func (p *printer) printNode(name string, attrs ...any) {
	indent := strings.Repeat("\t", p.depth)
	_, p.err = fmt.Fprint(p.w, indent, name)
	if p.err != nil {
		return
	}

	for i := 0; i < len(attrs); i++ {
		key := attrs[i]
		var v any
		if i+1 < len(attrs) {
			i++
			v = attrs[i]
		}

		_, p.err = fmt.Fprintf(p.w, " %s=%#v", key, v)
		if p.err != nil {
			return
		}
	}
}

func PrintAST(w io.Writer, root Node) error {
	p := &printer{w: w}
	Walk(root, p)
	return p.Err()
}
