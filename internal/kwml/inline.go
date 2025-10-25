package kwml

import (
	"slices"
	"unicode"
	"unicode/utf8"
)

type delimiter struct {
	literal string
	node    int
}

type runeStream struct {
	raw                  []byte
	index                int
	prev, curr           rune
	prevWidth, currWidth int
}

func (r *runeStream) advance() (char rune, width int) {
	if r.isAtEnd() {
		return
	}

	r.prev = r.curr
	r.prevWidth = r.currWidth
	r.curr, r.currWidth = utf8.DecodeRune(r.raw[r.index:])
	r.index += r.currWidth
	return r.curr, r.currWidth
}

func (r *runeStream) peek() (char rune, width int) {
	if r.isAtEnd() {
		return
	}

	return utf8.DecodeRune(r.raw[r.index:])
}

func (r *runeStream) match(char rune) bool {
	if r.isAtEnd() {
		return false
	}

	next, _ := r.peek()
	return next == char
}

func (r *runeStream) current() (char rune, width int) {
	return r.curr, r.currWidth
}

func (r *runeStream) currentBytes() []byte {
	i := r.index
	h := i - r.currWidth
	return r.raw[h:i]
}

func (r *runeStream) previous() (char rune, width int) {
	return r.prev, r.prevWidth
}

func (r *runeStream) isAtEnd() bool {
	return r.index >= len(r.raw)
}

func (r *runeStream) isEmpty() bool {
	return len(r.raw) == 0
}

func newRuneStream(raw []byte) *runeStream {
	return &runeStream{raw: raw}
}

type inlineParser struct {
	input      *runeStream
	buf        []byte
	delimiters []delimiter
	nodes      []InlineNode
	inVerbatim bool
}

func (i *inlineParser) parse() []InlineNode {
	// Special case: an empty input yields one empty text node.
	if i.input.isEmpty() {
		i.push(&Text{Content: i.input.raw})
		return i.nodes
	}

	for !i.input.isAtEnd() {
		char, width := i.input.advance()

		switch char {
		case '_':
			i.handleEmphasis("_", func(children []InlineNode) InlineNode {
				return &Emphasis{Children: children}
			})
		case '*':
			i.handleEmphasis("*", func(children []InlineNode) InlineNode {
				return &StrongEmphasis{Children: children}
			})
		default:
			i.growBuffer(width)
		}
	}

	i.commitBuffer()
	return i.nodes
}

func (i *inlineParser) handleEmphasis(literal string, produce func(children []InlineNode) InlineNode) {
	i.commitBuffer()

	spaceBefore, spaceAfter := i.detectWhitespace()
	canClose := !spaceBefore
	canOpen := !spaceAfter

	if canClose {
		if idx, delim := i.matchDelimiter(literal); idx != -1 {
			if c := i.nodes[delim.node+1:]; len(c) > 0 {
				children := make([]InlineNode, len(c))
				copy(children, c)

				i.nodes = append(i.nodes[:delim.node], produce(children))
				i.delimiters = i.delimiters[:idx]
				return
			}
		}
	}

	i.push(&Text{Content: i.input.currentBytes()})
	if canOpen {
		i.delimiters = append(i.delimiters, delimiter{
			literal: literal,
			node:    len(i.nodes) - 1,
		})
	}
}

func (i *inlineParser) matchDelimiter(pattern string) (index int, delim delimiter) {
	for index, delim = range slices.Backward(i.delimiters) {
		if delim.literal == pattern {
			return index, delim
		}
	}

	return -1, delim
}

func (i *inlineParser) detectWhitespace() (spaceBefore bool, spaceAfter bool) {
	prev, _ := i.input.previous()
	next, _ := i.input.peek()

	return unicode.IsSpace(prev), unicode.IsSpace(next)
}

func (i *inlineParser) growBuffer(n int) {
	i.buf = i.buf[:len(i.buf)+n]
}

func (i *inlineParser) commitBuffer() (committed bool) {
	if len(i.buf) > 0 {
		i.push(&Text{Content: i.buf})
		committed = true
	}

	i.buf = i.input.raw[i.input.index:i.input.index]
	return committed
}

func (i *inlineParser) push(node InlineNode) {
	i.nodes = append(i.nodes, node)
}

func newInlineParser(text []byte) *inlineParser {
	return &inlineParser{
		input: newRuneStream(text),
		buf:   text[0:0],
	}
}

type inlineVisitor struct {
	spans []InlineNode
}

func (i *inlineVisitor) StartVisit(node Node) (v Visitor) {
	if t, ok := node.(*Text); ok {
		i.spans = newInlineParser(t.Content).parse()
		return nil
	}

	return i
}

func (i *inlineVisitor) EndVisit(node Node) {
	// The first and only child of each heading and paragraph is a [Text] node
	// that will have been parsed into `spans` in [inlineVisitor.StartVisit].
	switch v := node.(type) {
	case *Heading:
		v.Children = i.spans
	case *Paragraph:
		v.Children = i.spans
	}
}
