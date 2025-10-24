package kwml

import (
	"slices"
	"unicode"
	"unicode/utf8"
)

type delimiter struct {
	literal string
	index   int
}

type inlineParser struct {
	raw        []byte
	buf        []byte
	prev, curr rune
	currLen    int
	index      int
	delimiters []delimiter
	nodes      []InlineNode
	inVerbatim bool
}

func (i *inlineParser) parse() []InlineNode {
	// Special case: an empty slice yields one empty text node.
	if len(i.raw) == 0 {
		i.nodes = append(i.nodes, &Text{
			Content: i.raw,
		})

		return i.nodes
	}

scanLoop:
	for !i.isAtEnd() {
		switch i.advance() {
		case '_', '*':
			i.resetBuffer()
			literal := string(i.curr)
			canClose := !unicode.IsSpace(i.prev)
			if canClose {
				if idx, delim := i.matchDelimiter(literal); idx != -1 {
					c := i.copyNodesAfter(delim)

					var grp InlineNode
					if i.curr == '_' {
						grp = &Emphasis{Children: c}
					} else {
						grp = &StrongEmphasis{Children: c}
					}

					i.nodes = append(i.nodes[:delim.index], grp)
					i.delimiters = i.delimiters[:idx]
					continue scanLoop
				}
			}

			i.nodes = append(i.nodes, &Text{
				Content: i.raw[i.index : i.index+i.currLen],
			})
			canOpen := !unicode.IsSpace(i.peek())
			if canOpen {
				i.delimiters = append(i.delimiters, delimiter{
					literal: literal,
					index:   len(i.nodes) - 1,
				})
			}
		default:
			i.growBuffer()
		}
	}

	i.resetBuffer()
	return i.nodes
}

func (i *inlineParser) matchDelimiter(pattern string) (index int, delim delimiter) {
	for index, delim = range slices.Backward(i.delimiters) {
		if delim.literal == pattern {
			return index, delim
		}
	}

	return -1, delimiter{}
}

func (i *inlineParser) copyNodesAfter(delim delimiter) []InlineNode {
	size := (len(i.nodes) - 1) - delim.index
	cpy := make([]InlineNode, size)
	copy(cpy, i.nodes[delim.index+1:])
	return cpy
}

func (i *inlineParser) resetBuffer() {
	if len(i.buf) > 0 {
		i.nodes = append(i.nodes, &Text{
			Content: i.buf,
		})
	}

	i.buf = i.raw[i.index:i.index]
}

func (i *inlineParser) growBuffer() {
	i.buf = i.buf[:len(i.buf)+i.currLen]
}

func (i *inlineParser) advance() (char rune) {
	if i.isAtEnd() {
		return 0
	}

	i.prev = i.curr
	i.curr, i.currLen = utf8.DecodeRune(i.raw[i.index:])
	i.index += i.currLen
	return i.curr
}

func (i *inlineParser) peek() (next rune) {
	if i.isAtEnd() {
		return 0
	}

	next, _ = utf8.DecodeRune(i.raw[i.index:])
	return next
}

func (i *inlineParser) isAtEnd() bool {
	return i.index >= len(i.raw)
}

func newInlineParser(text []byte) *inlineParser {
	return &inlineParser{
		raw:   text,
		buf:   text[0:0],
		nodes: []InlineNode{},
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
