package kwml

import (
	"slices"
)

type normalizer struct{}

func (n *normalizer) StartVisit(node Node) Visitor {
	switch v := node.(type) {
	case *Heading:
		v.Children = newInlineNormalizer(v.Children).normalize()
	case *Paragraph:
		v.Children = newInlineNormalizer(v.Children).normalize()
	case *Emphasis:
		v.Children = newInlineNormalizer(v.Children).normalize()
	case *StrongEmphasis:
		v.Children = newInlineNormalizer(v.Children).normalize()
	case *Link:
		v.Children = newInlineNormalizer(v.Children).normalize()
	}

	return n
}

func (n *normalizer) EndVisit(node Node) {}

type inlineNormalizer struct {
	input, normal []InlineNode
	index         int
}

func (i *inlineNormalizer) normalize() []InlineNode {
	for !i.isAtEnd() {
		node := i.advance()
		if t, ok := node.(*Text); ok {
			i.handleText(t)
		} else {
			i.commit(node)
		}
	}

	return i.normal
}

func (i *inlineNormalizer) handleText(text *Text) {
	next := i.peek()
	if next == nil {
		i.commit(text)
		return
	}

	if nextText, ok := next.(*Text); ok {
		buf := slices.Concat(text.Content, nextText.Content)
		i.advance()

		for !i.isAtEnd() {
			next = i.advance()
			if t, ok := next.(*Text); ok {
				buf = append(buf, t.Content...)
			} else {
				break
			}
		}

		i.commit(&Text{Content: buf})
	} else {
		i.commit(text)
	}
}

func (i *inlineNormalizer) commit(node InlineNode) {
	i.normal = append(i.normal, node)
}

func (i *inlineNormalizer) advance() InlineNode {
	if i.isAtEnd() {
		return nil
	}

	node := i.input[i.index]
	i.index++
	return node
}

func (i *inlineNormalizer) peek() InlineNode {
	if i.isAtEnd() {
		return nil
	} else {
		return i.input[i.index]
	}
}

func (i *inlineNormalizer) isAtEnd() bool {
	return i.index >= len(i.input)
}

func newInlineNormalizer(nodes []InlineNode) *inlineNormalizer {
	return &inlineNormalizer{
		input:  nodes,
		normal: make([]InlineNode, 0, len(nodes)),
	}
}
