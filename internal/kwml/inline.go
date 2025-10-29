package kwml

import (
	"slices"
	"strings"
	"unicode"
)

type delimiter struct {
	active  bool
	literal string
	node    int
}

type inlineParser struct {
	scanner    *scanner
	buf        []byte
	delimiters []*delimiter
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
		case '`':
			i.handleCode()
		case '[':
			i.handleOpenBracket()
		case ']':
			i.handleCloseBracket()
		default:
			i.growBuffer(width)
		}
	}

	i.flushText()
	return i.nodes
}

func (i *inlineParser) handleEmphasis(literal string, produce func(children []InlineNode) InlineNode) {
	i.flushText()
	i.resetBuffer()

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
		i.delimiters = append(i.delimiters, &delimiter{
			active:  true,
			literal: literal,
			node:    len(i.nodes) - 1,
		})
	}
}

func (i *inlineParser) matchDelimiter(pattern string) (index int, delim *delimiter) {
	for index, delim = range slices.Backward(i.delimiters) {
		if delim.active && delim.literal == pattern {
			return index, delim
		}
	}

	return -1, delim
}

func (i *inlineParser) deactivateDelimiters(pattern string) {
	for _, delim := range i.delimiters {
		if delim.literal == pattern {
			delim.active = false
		}
	}
}

func (i *inlineParser) detectWhitespace() (spaceBefore bool, spaceAfter bool) {
	prev, _ := i.scanner.previous()
	next, _ := i.scanner.peek()

	return unicode.IsSpace(prev), unicode.IsSpace(next)
}

func (i *inlineParser) handleCode() {
	i.flushText()

	// Minimum of one ` character needed to close the code span.
	minBackticks := 1
	for i.scanner.match('`') {
		i.scanner.advance()
		minBackticks++
	}

	i.resetBuffer()

	for !i.scanner.isAtEnd() {
		char, size := i.scanner.advance()
		if char == '`' {
			// Once we hit a backtick, scan up to `minBackticks` number of backtick
			// characters.
			numBackticks := 1
			for i.scanner.match('`') && numBackticks < minBackticks {
				i.scanner.advance()
				numBackticks++
			}

			if numBackticks == minBackticks {
				// We found the end of the code span, stop this scanning loop.
				break
			} else {
				// We didn't find the required number of backticks, so add them to the code
				// buffer instead.
				i.growBuffer(numBackticks)
			}
		} else {
			// Every other character is added to the code buffer.
			i.growBuffer(size)
		}
	}

	i.push(&Code{Raw: i.buf})
	i.resetBuffer()
}

func (i *inlineParser) handleOpenBracket() {
	i.flushText()
	i.resetBuffer()
	i.deactivateDelimiters("[")

	i.push(&Text{
		Content: i.scanner.currentBytes(),
	})
	i.delimiters = append(i.delimiters, &delimiter{
		active:  true,
		literal: "[",
		node:    len(i.nodes) - 1,
	})
}

func (i *inlineParser) handleCloseBracket() {
	// Mark the start point in case we discover this is an invalid link and need
	// to interpret all the scanned characters as text.
	start := i.scanner.index - 1

	if !i.scanner.match('(') {
		// Closing bracket wasn't followed by a target, just consume it
		// and move on.
		i.growBuffer(1)
		return
	}

	// Consume '('
	i.scanner.advance()

	idx, delim := i.matchDelimiter("[")
	if idx == -1 {
		// No matching delimiter, consume the "](" and move on.
		i.growBuffer(2)
		return
	}

	target := new(strings.Builder)
	valid := false
scanLoop:
	for !i.scanner.isAtEnd() {
		char, _ := i.scanner.advance()

		switch char {
		case '\\':
			if next, _ := i.scanner.peek(); canBackslashEscape(next) {
				i.scanner.advance()
				target.WriteRune(next)
			} else {
				target.WriteRune(char)
			}
		case '\n':
			break scanLoop
		case ')':
			valid = true
			break scanLoop
		default:
			target.WriteRune(char)
		}
	}

	if !valid {
		// Invalid link, consume everything we scanned as plain text and move on.
		i.growBuffer(i.scanner.index - start)
		return
	}

	i.flushText()
	c := i.nodes[delim.node+1:]
	link := &Link{
		Target:   target.String(),
		Children: make([]InlineNode, len(c)),
	}
	copy(link.Children, c)

	i.nodes = append(i.nodes[:delim.node], link)
	i.delimiters = i.delimiters[:idx]
	i.resetBuffer()
}

func (i *inlineParser) growBuffer(n int) {
	i.buf = i.buf[:len(i.buf)+n]
}

func (i *inlineParser) resetBuffer() {
	i.buf = i.scanner.cursor()
}

func (i *inlineParser) flushText() {
	if len(i.buf) > 0 {
		i.push(&Text{Content: i.buf})
	}
}

func (i *inlineParser) push(node InlineNode) {
	i.nodes = append(i.nodes, node)
}

func newInlineParser(text []byte) *inlineParser {
	scanner := newScanner(text)
	return &inlineParser{
		scanner: scanner,
		buf:     scanner.cursor(),
	}
}

func canBackslashEscape(char rune) bool {
	return char <= unicode.MaxASCII && unicode.IsPunct(char)
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
