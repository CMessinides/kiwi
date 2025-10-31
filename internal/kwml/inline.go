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
	active     bool
	literal    string
	startIndex int
	startAnnot *annotation
	midIndex   int
	midAnnot   *annotation
}

func openLinkish(literal string, index int, annot *annotation) *linkishOpener {
	return &linkishOpener{
		active:     true,
		literal:    literal,
		startIndex: index,
		startAnnot: annot,
		midIndex:   -1,
		midAnnot:   nil,
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

var reSpecialChar = regexp.MustCompile(`[*_()![\]\x60\n]`)

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
		spMatch := reSpecialChar.FindIndex(i.raw[pos:])

		if spMatch == nil {
			// No more special characters, skip to end.
			i.insertAnnotation(tagStr, pos, endpos)
			break
		}

		// At this point, we've found a special char.
		start := pos + spMatch[0]
		end := pos + spMatch[1]

		if start > pos {
			// Insert any leading text before the special character.
			i.insertAnnotation(tagStr, pos, start)
			pos = start
		}

		switch char := string(i.raw[start:end]); char {
		case "*", "_":
			var openTag, closeTag tag
			if char == "_" {
				openTag = tagOpenEmph
				closeTag = tagCloseEmph
			} else {
				openTag = tagOpenStrong
				closeTag = tagCloseStrong
			}

			prevChar, _ := utf8.DecodeLastRune(i.raw[:start])
			nextChar, _ := utf8.DecodeRune(i.raw[end:])

			canClose := !unicode.IsSpace(prevChar)
			canOpen := !unicode.IsSpace(nextChar)

			if canClose {
				if _, o := i.matchBalancedOpener(char); o != nil {
					zeroLength := o.annot.end == start
					if !zeroLength {
						o.deactivate()
						o.annot.tag = openTag
						c := i.insertAnnotation(closeTag, start, end)
						i.invalidateOpenersBetween(o.annot, c)
						pos = end
						continue
					}
				}
			}

			index := len(i.annotations)
			annot := i.insertAnnotation(tagStr, start, end)
			if canOpen {
				i.pushOpener(openBalanced(char, index, annot))
			}
			pos = end
		default:
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

func (i *inlineParser) nodes() []InlineNode {
	fmt.Printf("text: %s\n", i.raw)
	for idx, a := range i.annotations {
		fmt.Printf("  annotations[%3d] = %12s(%3d,%3d) %q\n", idx, a.tag, a.start, a.end, string(i.raw[a.start:a.end]))
	}
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
		fmt.Printf("toNodes(): %d remaining\n", len(rem))
		for i, a := range rem {
			fmt.Printf("  rem[%3d] = %12s(%3d,%3d) %q\n", i, a.tag, a.start, a.end, raw[a.start:a.end])
		}
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
		case tagCloseEmph, tagCloseStrong:
			return nodes, rem
		default:
			panic(fmt.Sprintf("unexpected annotation: %s", rem[0].tag))
		}
	}

	return nodes, rem
}

func toText(raw []byte, annotations []*annotation) (txt *Text, rem []*annotation) {
	start := annotations[0].start
	end := annotations[0].end

	for i := 1; i < len(annotations); i++ {
		annot := annotations[i]
		if annot.tag == tagStr {
			end = annot.end
		} else {
			rem = annotations[i:]
			break
		}
	}

	return &Text{
		Content: raw[start:end],
	}, rem
}

func toEmphasis(raw []byte, annotations []*annotation) (emph *Emphasis, rem []*annotation) {
	children, rem := toNodesUntil(tagCloseEmph, raw, annotations)
	return &Emphasis{Children: children}, rem
}

func toStrong(raw []byte, annotations []*annotation) (emph *StrongEmphasis, rem []*annotation) {
	children, rem := toNodesUntil(tagCloseStrong, raw, annotations)
	return &StrongEmphasis{Children: children}, rem
}

func toNodesUntil(closer tag, raw []byte, annotations []*annotation) (nodes []InlineNode, rem []*annotation) {
	nodes, rem = toNodes(raw, annotations)

	if len(rem) == 0 {
		panic(fmt.Sprintf("unclosed span: expected %s, got nil", closer))
	}

	if rem[0].tag != closer {
		panic(fmt.Sprintf("unclosed span: expected %s, got: %s", closer, rem[0].tag))
	}

	return nodes, rem[1:]
}
