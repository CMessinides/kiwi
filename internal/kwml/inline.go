package kwml

import (
	"fmt"
	"regexp"
	"slices"
	"unicode"
	"unicode/utf8"
)

type tag int

const (
	tagStr tag = iota
	tagOpenEmph
	tagCloseEmph
	tagOpenStrong
	tagCloseStrong
	tagOpenCode
	tagCloseCode
	tagOpenImageText
	tagCloseImageText
	tagOpenLinkText
	tagCloseLinkText
	tagOpenDest
	tagCloseDest
	tagOpenWikiLink
	tagCloseWikiLink
)

func (t tag) String() string {
	switch t {
	case tagStr:
		return "str"
	case tagOpenEmph:
		return "open_emph"
	case tagCloseEmph:
		return "close_emph"
	case tagOpenStrong:
		return "open_strong"
	case tagCloseStrong:
		return "close_strong"
	case tagOpenCode:
		return "open_code"
	case tagCloseCode:
		return "close_code"
	case tagOpenImageText:
		return "open_image_text"
	case tagCloseImageText:
		return "close_image_text"
	case tagOpenLinkText:
		return "open_link_text"
	case tagCloseLinkText:
		return "close_link_text"
	case tagOpenDest:
		return "open_dest"
	case tagCloseDest:
		return "close_dest"
	case tagOpenWikiLink:
		return "open_wiki_link"
	case tagCloseWikiLink:
		return "close_wiki_link"
	default:
		return "unknown"
	}
}

type annotation struct {
	tag        tag
	start, end int
}

func newAnnotation(tag tag, start int, end int) *annotation {
	return &annotation{
		tag:   tag,
		start: start,
		end:   end,
	}
}

type opener interface {
	isBetween(start int, end int) bool
	isActive() bool
	deactivate()
}

type balancedOpener struct {
	active  bool
	literal string
	index   int
	annot   *annotation
}

func openBalanced(literal string, index int, annot *annotation) *balancedOpener {
	return &balancedOpener{
		active:  true,
		literal: literal,
		index:   index,
		annot:   annot,
	}
}

type linkishOpener struct {
	active         bool
	literal        string
	startIndex     int
	startAnnot     *annotation
	midIndex       int
	midAnnot       *annotation
	textEndIndex   int
	textEndAnnot   *annotation
	destStartIndex int
	destStartAnnot *annotation
}

func openLinkish(
	literal string,
	startIndex int, startAnnot *annotation,
	textEndIndex int, textEndAnnot *annotation,
	destStartIndex int, destStartAnnot *annotation,
) *linkishOpener {
	return &linkishOpener{
		active:         true,
		literal:        literal,
		startIndex:     startIndex,
		startAnnot:     startAnnot,
		textEndIndex:   textEndIndex,
		textEndAnnot:   textEndAnnot,
		destStartIndex: destStartIndex,
		destStartAnnot: destStartAnnot,
	}
}

// Implement [opener] for all concrete opener types.

func (b *balancedOpener) isActive() bool { return b.active }
func (b *balancedOpener) deactivate()    { b.active = false }

func (b *balancedOpener) isBetween(start int, end int) bool {
	return b.annot.start >= start && b.annot.end <= end
}

func (l *linkishOpener) isActive() bool { return l.active }
func (l *linkishOpener) deactivate()    { l.active = false }

func (l *linkishOpener) isBetween(start int, end int) bool {
	startBetween := l.startAnnot.start >= start && l.startAnnot.end <= end

	if l.midAnnot == nil {
		return startBetween
	} else {
		midBetween := l.midAnnot.start >= start && l.midAnnot.end <= end
		return startBetween || midBetween
	}
}

type inlineParser struct {
	raw         []byte
	annotations []*annotation
	openers     []opener
}

var (
	reSpecialChar      = regexp.MustCompile(`[*_()![\]\x60\n]`)
	reBackticks        = regexp.MustCompile(`\x60+`)
	reLeadingBackticks = regexp.MustCompile(`^\x60+`)
)

type matcher func(parser *inlineParser, start, end int) (pos int)

func emphasisMatcher(literal string, openTag tag, closeTag tag) matcher {
	return func(parser *inlineParser, start, end int) (pos int) {
		prevChar, _ := utf8.DecodeLastRune(parser.raw[:start])
		nextChar, _ := utf8.DecodeRune(parser.raw[end:])

		canClose := !unicode.IsSpace(prevChar)
		canOpen := !unicode.IsSpace(nextChar)

		if canClose {
			if _, o := parser.matchBalancedOpener(literal); o != nil {
				zeroLength := o.annot.end == start
				if !zeroLength {
					o.deactivate()
					o.annot.tag = openTag
					c := parser.insertAnnotation(closeTag, start, end)
					parser.invalidateOpenersBetween(o.annot, c)
					return end
				}
			}
		}

		index := len(parser.annotations)
		annot := parser.insertAnnotation(tagStr, start, end)
		if canOpen {
			parser.pushOpener(openBalanced(literal, index, annot))
		}
		return end
	}
}

var (
	matchUnderscore = emphasisMatcher("_", tagOpenEmph, tagCloseEmph)
	matchStar       = emphasisMatcher("*", tagOpenStrong, tagCloseStrong)
)

var matchBacktick matcher = func(parser *inlineParser, start, end int) (pos int) {
	backticks := reLeadingBackticks.FindIndex(parser.raw[start:])
	count := backticks[1] - backticks[0]
	end += count - 1
	parser.insertAnnotation(tagOpenCode, start, end)

	start = end
	for end < len(parser.raw) {
		delim := reBackticks.FindIndex(parser.raw[end:])
		if delim == nil {
			end = len(parser.raw)
			parser.insertAnnotation(tagStr, start, end)
			parser.insertAnnotation(tagCloseCode, end, end)
			break
		}

		end += delim[0]
		n := delim[1] - delim[0]
		if n == count {
			parser.insertAnnotation(tagStr, start, end)
			start, end = end, end+n
			parser.insertAnnotation(tagCloseCode, start, end)
			break
		} else {
			end += n
		}
	}

	return end
}

var matchLeftBracket matcher = func(parser *inlineParser, start, end int) (pos int) {
	nextChar, size := utf8.DecodeRune(parser.raw[end:])
	if nextChar == '[' {
		end += size
	}

	index := len(parser.annotations)
	annot := parser.insertAnnotation(tagStr, start, end) // "[" or "[["
	literal := string(parser.raw[start:end])
	parser.pushOpener(openBalanced(literal, index, annot))
	return end
}

var matchRightBracket matcher = func(parser *inlineParser, start, end int) (pos int) {
	// Check if this is the first character of a "]]" pair.
	nextChar, size := utf8.DecodeRune(parser.raw[end:])
	if nextChar == ']' {
		end += size
		endIndex := len(parser.annotations)
		closer := parser.insertAnnotation(tagStr, start, end) // "]]"

		if i, b := parser.matchBalancedOpener("[["); b != nil {
			b.annot.tag = tagOpenWikiLink
			closer.tag = tagCloseWikiLink
			parser.invalidateOpenersBetween(b.annot, closer)
			parser.stringifyAnnotationsBetween(i+1, endIndex)
		}

		return end
	}

	parser.insertAnnotation(tagStr, start, end) // "]"
	if i, b := parser.matchBalancedOpener("["); b != nil {
		nextChar, size := utf8.DecodeRune(parser.raw[end:])
		if nextChar == '(' {
			start = end
			end = start + size
			parser.insertAnnotation(tagStr, start, end) // "("

			textEndIndex := len(parser.annotations) - 2
			destStartIndex := textEndIndex + 1

			// Invalidate any openers between "[" and "](" -- even if this doesn't turn
			// out to be a link, it avoids unexpected situations where, for example, an
			// underscore in a URL creates an emphasis span opened by an underscore in
			// the link text.
			startAnnot := b.annot
			textEndAnnot := parser.annotations[textEndIndex]
			parser.invalidateOpenersBetween(startAnnot, textEndAnnot)

			parser.openers[i] = openLinkish(
				b.literal,
				b.index,
				startAnnot, // "["
				textEndIndex,
				textEndAnnot, // "]"
				destStartIndex,
				parser.annotations[destStartIndex], // "("
			)
		} else {
			b.deactivate()
		}
	}

	return end
}

var matchLeftParen matcher = func(parser *inlineParser, start, end int) (pos int) {
	index := len(parser.annotations)
	annot := parser.insertAnnotation(tagStr, start, end)
	parser.pushOpener(openBalanced("(", index, annot))
	return end
}

var matchRightParen matcher = func(parser *inlineParser, start, end int) (pos int) {
	if _, b := parser.matchBalancedOpener("("); b != nil {
		b.deactivate()
		parser.insertAnnotation(tagStr, start, end)
	} else if _, l := parser.matchLinkishOpener(); l != nil {
		l.startAnnot.tag = tagOpenLinkText
		l.textEndAnnot.tag = tagCloseLinkText
		l.destStartAnnot.tag = tagOpenDest
		destEndIndex := len(parser.annotations)
		destEndAnnot := parser.insertAnnotation(tagCloseDest, start, end)

		parser.invalidateOpenersBetween(l.destStartAnnot, destEndAnnot)
		parser.stringifyAnnotationsBetween(l.destStartIndex+1, destEndIndex)
	}

	return end
}

var matchers = map[string]matcher{
	"_": matchUnderscore,
	"*": matchStar,
	"`": matchBacktick,
	"[": matchLeftBracket,
	"]": matchRightBracket,
	"(": matchLeftParen,
	")": matchRightParen,
}

func (i *inlineParser) parse(text []byte) {
	i.reset(text)

	if len(i.raw) == 0 {
		// Special case: an empty input yields one empty text node.
		i.insertAnnotation(tagStr, 0, 0)
		return
	}

	pos := 0
	endpos := len(i.raw)
	for pos < endpos {
		match := reSpecialChar.FindIndex(i.raw[pos:])

		if match == nil {
			// No more special characters, skip to end.
			i.insertAnnotation(tagStr, pos, endpos)
			break
		}

		// At this point, we've found a special char.
		start := pos + match[0]
		end := pos + match[1]

		if start > pos {
			// Insert any leading text before the special character.
			i.insertAnnotation(tagStr, pos, start)
			pos = start
		}

		if matcher, ok := matchers[string(i.raw[start:end])]; ok {
			pos = matcher(i, start, end)
		} else {
			i.insertAnnotation(tagStr, start, end)
			pos = end
		}
	}
}

func (i *inlineParser) pushOpener(o opener) {
	i.openers = append(i.openers, o)
}

func (i *inlineParser) matchBalancedOpener(literal string) (index int, opener *balancedOpener) {
	for i, o := range slices.Backward(i.openers) {
		if o.isActive() {
			if b, ok := o.(*balancedOpener); ok {
				if b.literal == literal {
					return i, b
				}
			}
		}
	}

	return -1, nil
}

func (i *inlineParser) matchLinkishOpener() (index int, opener *linkishOpener) {
	for i, o := range slices.Backward(i.openers) {
		if o.isActive() {
			if l, ok := o.(*linkishOpener); ok {
				return i, l
			}
		}
	}

	return -1, nil
}

func (i *inlineParser) invalidateOpenersBetween(from *annotation, to *annotation) {
	for _, o := range i.openers {
		if !o.isActive() {
			continue
		}

		if o.isBetween(from.end, to.start) {
			o.deactivate()
		}
	}
}

func (i *inlineParser) insertAnnotation(tag tag, start int, end int) *annotation {
	annot := newAnnotation(tag, start, end)
	i.annotations = append(i.annotations, annot)
	return annot
}

func (i *inlineParser) stringifyAnnotationsBetween(start, end int) {
	for _, annot := range i.annotations[start:end] {
		annot.tag = tagStr
	}
}

func (i *inlineParser) nodes() []InlineNode {
	nodes, rem := toNodes(i.raw, i.annotations)
	if len(rem) != 0 {
		panic(fmt.Sprintf("expected no remaining annotations; got %d", len(rem)))
	}

	return nodes
}

func (i *inlineParser) reset(raw []byte) {
	i.raw = raw
	i.annotations = nil
	i.openers = nil
}

func (i *inlineParser) StartVisit(node Node) (v Visitor) {
	if t, ok := node.(*Text); ok {
		i.parse(t.Content)
		return nil
	} else {
		return i
	}
}

func (i *inlineParser) EndVisit(node Node) {
	// The first and only child of each heading and paragraph is a [Text] node
	// that will have been parsed in [inlineParser.StartVisit].
	switch v := node.(type) {
	case *Heading:
		v.Children = i.nodes()
	case *Paragraph:
		v.Children = i.nodes()
	}
}

func newInlineParser() *inlineParser {
	return &inlineParser{}
}

func canBackslashEscape(char rune) bool {
	return char <= unicode.MaxASCII && unicode.IsPunct(char)
}

func toNodes(raw []byte, annotations []*annotation) (nodes []InlineNode, rem []*annotation) {
	rem = annotations

	for len(rem) > 0 {
		switch rem[0].tag {
		case tagStr:
			var txt *Text
			txt, rem = toText(raw, rem)
			nodes = append(nodes, txt)
		case tagOpenEmph:
			var emph *Emphasis
			emph, rem = toEmphasis(raw, rem[1:])
			nodes = append(nodes, emph)
		case tagOpenStrong:
			var strong *StrongEmphasis
			strong, rem = toStrong(raw, rem[1:])
			nodes = append(nodes, strong)
		case tagOpenCode:
			var code *Code
			code, rem = toCode(raw, rem)
			nodes = append(nodes, code)
		case tagOpenLinkText:
			var link *Link
			link, rem = toLink(raw, rem[1:])
			nodes = append(nodes, link)
		case tagOpenWikiLink:
			var link *WikiLink
			link, rem = toWikiLink(raw, rem)
			nodes = append(nodes, link)
		case tagCloseEmph, tagCloseStrong, tagCloseCode, tagCloseLinkText, tagCloseWikiLink:
			return nodes, rem
		default:
			panic(fmt.Sprintf("unexpected annotation: %s", rem[0].tag))
		}
	}

	return nodes, rem
}

func toText(raw []byte, annotations []*annotation) (txt *Text, rem []*annotation) {
	start, end := annotations[0].start, annotations[0].end
	for i, annot := range annotations {
		if annot.tag == tagStr {
			end = annot.end
		} else {
			return &Text{Content: raw[start:end]}, annotations[i:]
		}
	}

	return &Text{Content: raw[start:end]}, nil
}

func toCode(raw []byte, annotations []*annotation) (code *Code, rem []*annotation) {
	start, end := annotations[0].end, annotations[0].end
	for i, annot := range annotations {
		if annot.tag != tagCloseCode {
			end = annot.end
		} else {
			return &Code{Raw: raw[start:end]}, annotations[i+1:]
		}
	}

	return &Code{Raw: raw[start:end]}, nil
}

func toEmphasis(raw []byte, annotations []*annotation) (emph *Emphasis, rem []*annotation) {
	children, rem := toNodesUntil(tagCloseEmph, raw, annotations)
	return &Emphasis{Children: children}, rem
}

func toStrong(raw []byte, annotations []*annotation) (emph *StrongEmphasis, rem []*annotation) {
	children, rem := toNodesUntil(tagCloseStrong, raw, annotations)
	return &StrongEmphasis{Children: children}, rem
}

func toWikiLink(raw []byte, annotations []*annotation) (link *WikiLink, rem []*annotation) {
	start, end := annotations[0].end, annotations[0].end
	for i, annot := range annotations {
		if annot.tag != tagCloseWikiLink {
			end = annot.end
		} else {
			return &WikiLink{Target: string(raw[start:end])}, annotations[i+1:]
		}
	}

	panic(fmt.Sprintf("unclosed wiki link: expected %s, got nil", tagCloseWikiLink))
}

func toLink(raw []byte, annotations []*annotation) (link *Link, rem []*annotation) {
	children, rem := toNodesUntil(tagCloseLinkText, raw, annotations)
	destStart, rem := consume(tagOpenDest, rem)

	targetStart, targetEnd := destStart.end, destStart.end
	for i, annot := range rem {
		if annot.tag != tagCloseDest {
			targetEnd = annot.end
		} else {
			return &Link{
				Children: children,
				Target:   string(raw[targetStart:targetEnd]),
			}, rem[i+1:]
		}
	}

	panic(fmt.Sprintf("unclosed link target: expected %s, got nil", tagCloseDest))
}

func toNodesUntil(closer tag, raw []byte, annotations []*annotation) (nodes []InlineNode, rem []*annotation) {
	nodes, rem = toNodes(raw, annotations)
	_, rem = consume(closer, rem)
	return nodes, rem
}

func consume(expected tag, annotations []*annotation) (annot *annotation, rem []*annotation) {
	if len(annotations) == 0 {
		panic(fmt.Sprintf("expected %s, got nil", expected))
	} else if annotations[0].tag != expected {
		panic(fmt.Sprintf("expected %s, got: %s", expected, annotations[0].tag))
	} else {
		return annotations[0], annotations[1:]
	}
}
