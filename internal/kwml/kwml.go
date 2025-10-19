// Package kwml provides a parser for the [K]i[W]i [M]arkup [L]anguage, a
// lightweight markup language for content in kiwi wikis. It's inspired by
// Markdown, but adds support for Wikilinks, transclusion, and flexible
// sections called "blocks", which allow embedded DSLs to extend the final
// HTML output.
package kwml

import "slices"

type Node interface {
	Children() []Node
}

type NodeAppender interface {
	Node
	Append(children ...Node) NodeAppender
}

type LineAppender interface {
	Node
	AppendLine(line []byte) LineAppender
}

type Document struct {
	Blocks []Node
}

func (d *Document) Children() []Node {
	return d.Blocks
}

func (d *Document) Append(children ...Node) NodeAppender {
	d.Blocks = append(d.Blocks, children...)
	return d
}

type Heading struct {
	Level int
	Spans []Node
}

func (h *Heading) Children() []Node {
	return h.Spans
}

func (h *Heading) Append(children ...Node) NodeAppender {
	h.Spans = append(h.Spans, children...)
	return h
}

type Paragraph struct {
	Spans []Node
}

func (p *Paragraph) Children() []Node {
	return p.Spans
}

func (p *Paragraph) Append(children ...Node) NodeAppender {
	p.Spans = append(p.Spans, children...)
	return p
}

type Blockquote struct {
	Content []Node
}

func (o *Blockquote) Children() []Node {
	return o.Content
}

func (o *Blockquote) Append(children ...Node) NodeAppender {
	o.Content = append(o.Content, children...)
	return o
}

type OrderedList struct {
	Items []Node
}

func (o *OrderedList) Children() []Node {
	return o.Items
}

func (o *OrderedList) Append(children ...Node) NodeAppender {
	o.Items = append(o.Items, children...)
	return o
}

type UnorderedList struct {
	Items []Node
}

func (u *UnorderedList) Append(children ...Node) NodeAppender {
	u.Items = append(u.Items, children...)
	return u
}

func (u *UnorderedList) Children() []Node {
	return u.Items
}

type ListItem struct {
	Content []Node
}

func (l *ListItem) Children() []Node {
	return l.Content
}

func (l *ListItem) Append(children ...Node) NodeAppender {
	l.Content = append(l.Content, children...)
	return l
}

type Verbatim struct {
	Lang    string
	Content []byte
}

func (v *Verbatim) Children() []Node {
	return nil
}

func (v *Verbatim) AppendLine(line []byte) LineAppender {
	if len(v.Content) == 0 {
		v.Content = slices.Clone(line)
	} else {
		v.Content = append(v.Content, '\n')
		v.Content = append(v.Content, line...)
	}

	return v
}

type Macro struct {
	Tag     string
	Content []byte
}

func (m *Macro) Children() []Node {
	return nil
}

func (m *Macro) AppendLine(line []byte) LineAppender {
	if len(m.Content) == 0 {
		m.Content = slices.Clone(line)
	} else {
		m.Content = append(m.Content, '\n')
		m.Content = append(m.Content, line...)
	}

	return m
}

type Text struct {
	Content []byte
}

func (t *Text) Children() []Node {
	return nil
}

func (t *Text) AppendLine(line []byte) LineAppender {
	if len(t.Content) == 0 {
		t.Content = slices.Clone(line)
	} else {
		t.Content = append(t.Content, '\n')
		t.Content = append(t.Content, line...)
	}

	return t
}

type Emphasis struct {
	Spans []Node
}

func (e *Emphasis) Children() []Node {
	return e.Spans
}

func (e *Emphasis) Append(children ...Node) NodeAppender {
	e.Spans = append(e.Spans, children...)
	return e
}

type StrongEmphasis struct {
	Spans []Node
}

func (s *StrongEmphasis) Children() []Node {
	return s.Spans
}

func (s *StrongEmphasis) Append(children ...Node) NodeAppender {
	s.Spans = append(s.Spans, children...)
	return s
}

type Code struct {
	Content []byte
}

func (c *Code) Children() []Node {
	return nil
}

type Link struct {
	Spans  []Node
	Target string
}

func (l *Link) Children() []Node {
	return l.Spans
}

func (l *Link) Append(children ...Node) NodeAppender {
	l.Spans = append(l.Spans, children...)
	return l
}

type WikiLink struct {
	Target string
}

func (w *WikiLink) Children() []Node {
	return nil
}

type Visitor interface {
	StartVisit(node Node) (v Visitor)
	EndVisit(node Node)
}

func Walk(root Node, v Visitor) {
	if v = v.StartVisit(root); v == nil {
		return
	}

	for _, c := range root.Children() {
		Walk(c, v)
	}

	v.EndVisit(root)
}
