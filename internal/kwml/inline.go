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

type scanner struct {
	raw                  []byte
	index                int
	prev, curr           rune
	prevWidth, currWidth int
}

func (s *scanner) advance() (char rune, width int) {
	if s.isAtEnd() {
		return
	}

	s.prev = s.curr
	s.prevWidth = s.currWidth
	s.curr, s.currWidth = utf8.DecodeRune(s.raw[s.index:])
	s.index += s.currWidth
	return s.curr, s.currWidth
}

func (s *scanner) peek() (char rune, width int) {
	if s.isAtEnd() {
		return
	}

	return utf8.DecodeRune(s.raw[s.index:])
}

func (s *scanner) match(char rune) bool {
	if s.isAtEnd() {
		return false
	}

	next, _ := s.peek()
	return next == char
}

func (s *scanner) current() (char rune, width int) {
	return s.curr, s.currWidth
}

func (s *scanner) currentBytes() []byte {
	i := s.index
	h := i - s.currWidth
	return s.raw[h:i]
}

func (s *scanner) previous() (char rune, width int) {
	return s.prev, s.prevWidth
}

func (s *scanner) isAtEnd() bool {
	return s.index >= len(s.raw)
}

func (s *scanner) isEmpty() bool {
	return len(s.raw) == 0
}

func newScanner(raw []byte) *scanner {
	return &scanner{raw: raw}
}

type inlineParser struct {
	scanner    *scanner
	buf        []byte
	delimiters []delimiter
	nodes      []InlineNode
	inVerbatim bool
}

func (i *inlineParser) parse() []InlineNode {
	// Special case: an empty input yields one empty text node.
	if i.scanner.isEmpty() {
		i.push(&Text{Content: i.scanner.raw})
		return i.nodes
	}

	for !i.scanner.isAtEnd() {
		char, width := i.scanner.advance()

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

	i.push(&Text{Content: i.scanner.currentBytes()})
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
	prev, _ := i.scanner.previous()
	next, _ := i.scanner.peek()

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

	i.buf = i.scanner.raw[i.scanner.index:i.scanner.index]
	return committed
}

func (i *inlineParser) push(node InlineNode) {
	i.nodes = append(i.nodes, node)
}

func newInlineParser(text []byte) *inlineParser {
	return &inlineParser{
		scanner: newScanner(text),
		buf:     text[0:0],
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
