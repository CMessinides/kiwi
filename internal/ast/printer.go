package ast

import (
	"fmt"
	"io"
	"maps"
	"strings"
)

type (
	AttrStringerMap map[string]func(value any) fmt.Stringer
)

type DefaultAttrStringer struct {
	Value any
}

func (d DefaultAttrStringer) String() string {
	return fmt.Sprintf("%#v", d.Value)
}

type AttrStringerFunc func() string

func (a AttrStringerFunc) String() string {
	return a()
}

// Printer wraps an [io.Writer] to pretty-print [Node]s to it.
type Printer[T comparable] struct {
	w           io.Writer
	depth       int
	indentChars string
	includeKeys bool
	stringers   AttrStringerMap
}

// Print pretty-prints the `node` to the printer's [io.Writer]. It returns the
// first error it encounters, if any, while writing.
func (p *Printer[T]) Print(node *Node[T]) error {
	return p.VisitNode(node)
}

// VisitNode implements the [NodeVisitor] interface by pretty-printing `node`
// and its children recursively.
func (p *Printer[T]) VisitNode(node *Node[T]) error {
	indent := strings.Repeat(p.indentChars, p.depth)

	// Print the node type first.
	_, err := fmt.Fprint(p.w, indent, "(", node.Type)
	if err != nil {
		return err
	}

	attrs := node.Attrs()
	if attrs.Size() != 0 {
		err = p.printAttrs(attrs)
		if err != nil {
			return err
		}
	}

	children := node.Children()
	isLeaf := len(children) == 0
	if isLeaf {
		_, err = fmt.Fprint(p.w, ")")
		if err != nil {
			return err
		}
	} else {
		p.depth += 1
		defer func() { p.depth -= 1 }()

		for _, child := range children {
			_, err = fmt.Fprintf(p.w, "\n")
			if err != nil {
				return err
			}

			err = p.VisitNode(child)
			if err != nil {
				return err
			}

		}

		_, err = fmt.Fprint(p.w, ")")
	}

	return err
}

func (p *Printer[T]) printAttrs(attrs *AttributeMap) error {
	var err error

	for k, v := range attrs.Entries() {
		var s fmt.Stringer = DefaultAttrStringer{v}
		if f, ok := p.stringers[k]; ok {
			s = f(v)
		}

		if p.includeKeys {
			_, err = fmt.Fprint(p.w, " ", k, "=", s)
		} else {
			_, err = fmt.Fprint(p.w, " ", s)
		}

		if err != nil {
			return err
		}
	}

	return err
}

// PrinterOptions provides configuration for a [Printer].
type PrinterOptions struct {
	AttrStringers AttrStringerMap
	// Characters to use for indentation (defaults to "\t").
	IndentChars string
	// Whether to print attribute keys.
	IncludeAttrKeys bool
}

// NewPrinter creates a new [Printer] that wraps the given [io.Writer].
func NewPrinter[T comparable](w io.Writer, opts *PrinterOptions) *Printer[T] {
	p := &Printer[T]{
		w:           w,
		indentChars: "\t",
		stringers:   make(AttrStringerMap),
	}

	if opts.AttrStringers != nil {
		maps.Copy(p.stringers, opts.AttrStringers)
	}

	if opts.IndentChars != "" {
		p.indentChars = opts.IndentChars
	}

	if opts.IncludeAttrKeys {
		p.includeKeys = true
	}

	return p
}
