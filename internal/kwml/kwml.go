// Package kwml provides a parser for the [K]i[W]i [M]arkup [L]anguage, a
// lightweight markup language for content in kiwi wikis. It's inspired by
// Markdown, but adds support for Wikilinks, transclusion, and flexible
// sections called "blocks", which allow embedded DSLs to extend the final
// HTML output.
package kwml

import (
	"iter"
	"slices"
)

// Interfaces

type Node any

type BlockNode interface {
	Node
	blockNode()
}

type InlineNode interface {
	Node
	inlineNode()
}

type BlockContainer interface {
	Blocks() iter.Seq[BlockNode]
}

type InlineContainer interface {
	Inlines() iter.Seq[InlineNode]
}

type blockAppender interface {
	BlockNode
	appendBlocks(blocks ...BlockNode)
}

type inlineContainer interface {
	replaceInlines(nodes []InlineNode)
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

// Implement [BlockContainer] for block nodes with block children.

func (d *Document) Blocks() iter.Seq[BlockNode]   { return slices.Values(d.Children) }
func (b *Blockquote) Blocks() iter.Seq[BlockNode] { return slices.Values(b.Children) }
func (l *ListItem) Blocks() iter.Seq[BlockNode]   { return slices.Values(l.Children) }

func (o *OrderedList) Blocks() iter.Seq[BlockNode] {
	return func(yield func(BlockNode) bool) {
		for _, b := range o.Items {
			if !yield(b) {
				return
			}
		}
	}
}

func (u *UnorderedList) Blocks() iter.Seq[BlockNode] {
	return func(yield func(BlockNode) bool) {
		for _, b := range u.Items {
			if !yield(b) {
				return
			}
		}
	}
}

// Implement [InlineContainer] for block nodes with inline children.

func (h *Heading) Inlines() iter.Seq[InlineNode]   { return slices.Values(h.Children) }
func (p *Paragraph) Inlines() iter.Seq[InlineNode] { return slices.Values(p.Children) }

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

// Implement [inlineContainer] for block nodes that contain inline nodes.

func (p *Paragraph) replaceInlines(nodes []InlineNode) {
	p.Children = nodes
}

func (h *Heading) replaceInlines(nodes []InlineNode) {
	h.Children = nodes
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

// Implement [InlineContainer] for inline nodes with inline children.

func (e *Emphasis) Inlines() iter.Seq[InlineNode]       { return slices.Values(e.Children) }
func (s *StrongEmphasis) Inlines() iter.Seq[InlineNode] { return slices.Values(s.Children) }
func (l *Link) Inlines() iter.Seq[InlineNode]           { return slices.Values(l.Children) }

// Implement [inlineContainer] for inline nodes with children.

func (e *Emphasis) replaceInlines(nodes []InlineNode) {
	e.Children = nodes
}

func (s *StrongEmphasis) replaceInlines(nodes []InlineNode) {
	s.Children = nodes
}

func (l *Link) replaceInlines(nodes []InlineNode) {
	l.Children = nodes
}

// Implement [lineWriter] for text nodes.
func (t *Text) writeLine(line []byte) { t.Content = appendLine(t.Content, line) }
