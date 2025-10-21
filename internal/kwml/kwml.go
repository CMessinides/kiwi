// Package kwml provides a parser for the [K]i[W]i [M]arkup [L]anguage, a
// lightweight markup language for content in kiwi wikis. It's inspired by
// Markdown, but adds support for Wikilinks, transclusion, and flexible
// sections called "blocks", which allow embedded DSLs to extend the final
// HTML output.
package kwml

import (
	"iter"
)

// Interfaces

type Node interface{}

type BlockNode interface {
	Node
	blockNode()
}

type InlineNode interface {
	Node
	inlineNode()
}

type blockAppender interface {
	BlockNode
	appendBlocks(blocks ...BlockNode)
}

type listItemAppender interface {
	BlockNode
	appendListItems(items ...*ListItem)
}

type lineWriter interface {
	writeLine(line []byte)
}

type rawBlock interface {
	BlockNode
	lineWriter
}

type listBlock interface {
	BlockNode
	listItemAppender
}

func appendLine(curr []byte, line []byte) []byte {
	if len(curr) > 0 {
		curr = append(curr, '\n')
	}

	return append(curr, line...)
}

// Block nodes
type (
	Document struct {
		Children []BlockNode
	}

	Heading struct {
		Level    int
		Children []InlineNode
	}

	Paragraph struct {
		Children []InlineNode
	}

	Blockquote struct {
		Children []BlockNode
	}

	OrderedList struct {
		Items []*ListItem
	}

	UnorderedList struct {
		Items []*ListItem
	}

	ListItem struct {
		Children []BlockNode
	}

	Verbatim struct {
		Lang string
		Raw  []byte
	}

	Macro struct {
		Tag string
		Raw []byte
	}
)

// Implement [BlockNode] for all concrete block types.
func (d *Document) blockNode()      {}
func (h *Heading) blockNode()       {}
func (p *Paragraph) blockNode()     {}
func (b *Blockquote) blockNode()    {}
func (o *OrderedList) blockNode()   {}
func (u *UnorderedList) blockNode() {}
func (l *ListItem) blockNode()      {}
func (v *Verbatim) blockNode()      {}
func (m *Macro) blockNode()         {}

// Implement [childrenIterator] for block nodes with children.
func (d *Document) iterChildren() iter.Seq[Node]      { return blockIter(d.Children) }
func (h *Heading) iterChildren() iter.Seq[Node]       { return inlineIter(h.Children) }
func (p *Paragraph) iterChildren() iter.Seq[Node]     { return inlineIter(p.Children) }
func (b *Blockquote) iterChildren() iter.Seq[Node]    { return blockIter(b.Children) }
func (o *OrderedList) iterChildren() iter.Seq[Node]   { return listItemIter(o.Items) }
func (u *UnorderedList) iterChildren() iter.Seq[Node] { return listItemIter(u.Items) }
func (l *ListItem) iterChildren() iter.Seq[Node]      { return blockIter(l.Children) }

// Implement [blockAppender] for block nodes that contain other blocks.

func (d *Document) appendBlocks(blocks ...BlockNode) {
	d.Children = append(d.Children, blocks...)
}

func (b *Blockquote) appendBlocks(blocks ...BlockNode) {
	b.Children = append(b.Children, blocks...)
}

func (l *ListItem) appendBlocks(blocks ...BlockNode) {
	l.Children = append(l.Children, blocks...)
}

// Implement [listItemAppender] for list blocks.

func (o *OrderedList) appendListItems(items ...*ListItem) {
	o.Items = append(o.Items, items...)
}

func (u *UnorderedList) appendListItems(items ...*ListItem) {
	u.Items = append(u.Items, items...)
}

// Implement [lineWriter] for the raw blocks, verbatim and macro.
func (v *Verbatim) writeLine(line []byte) { v.Raw = appendLine(v.Raw, line) }
func (m *Macro) writeLine(line []byte)    { m.Raw = appendLine(m.Raw, line) }

// Inline nodes
type (
	Text struct {
		Content []byte
	}

	Emphasis struct {
		Children []InlineNode
	}

	StrongEmphasis struct {
		Children []InlineNode
	}

	Code struct {
		Raw []byte
	}

	Link struct {
		Target   string
		Children []InlineNode
	}

	WikiLink struct {
		Target string
	}
)

// Implement [InlineNode] for all concrete inline types.
func (t *Text) inlineNode()           {}
func (e *Emphasis) inlineNode()       {}
func (s *StrongEmphasis) inlineNode() {}
func (c *Code) inlineNode()           {}
func (l *Link) inlineNode()           {}
func (w *WikiLink) inlineNode()       {}

// Implement [childrenIterator] for inline nodes with children.
func (e *Emphasis) iterChildren() iter.Seq[Node]       { return inlineIter(e.Children) }
func (s *StrongEmphasis) iterChildren() iter.Seq[Node] { return inlineIter(s.Children) }
func (l *Link) iterChildren() iter.Seq[Node]           { return inlineIter(l.Children) }

// Implement [lineWriter] for text nodes.
func (t *Text) writeLine(line []byte) { t.Content = appendLine(t.Content, line) }
