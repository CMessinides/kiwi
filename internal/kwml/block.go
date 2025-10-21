package kwml

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
)

var (
	reBlankLine     = regexp.MustCompile(`^\s*$`)
	reNonBlankLine  = regexp.MustCompile(`^\s*(\S|\S.*\S)\s*$`)
	reHeadingPrefix = regexp.MustCompile(`^\s*(#{1,6})\s`)
	// /x60 = literal backtick
	reRawOpener              = regexp.MustCompile(`^\s*(\x60{3,})(@\w+|\w*)\s*$`)
	reBlockquoteOpener       = regexp.MustCompile(`^\s*("{3,})\s*$`)
	reOrderedListItemPrefix  = regexp.MustCompile(`^\s*\+\s`)
	reUnorderdListItemPrefix = regexp.MustCompile(`^\s*-\s`)
)

// findText returns all the characters between the first non-whitespace and
// last non-whitespace character in `line`. If the line is empty, or only
// contains whitespace, findText returns nil.
func findText(line []byte) []byte {
	if len(line) == 0 {
		return nil
	}

	if match := reNonBlankLine.FindSubmatch(line); match != nil {
		// The first submatch is just the text, with leading and trailing
		// whitespace trimmed.
		return match[1]
	} else {
		return nil
	}
}

// tryOpenParagraph tests if the line opens a paragraph. If it does, it returns
// the new node.
func tryOpenParagraph(line []byte) (para *Paragraph, text *Text) {
	if l := findText(line); l != nil {
		t := &Text{}
		t.writeLine(l)
		p := &Paragraph{
			Children: []InlineNode{t},
		}

		return p, t
	}

	return nil, nil
}

// tryOpenHeading tests if the line opens a heading. If it does, it returns the
// new node.
func tryOpenHeading(line []byte) *Heading {
	if idxs := reHeadingPrefix.FindSubmatchIndex(line); idxs != nil {
		level := idxs[3] - idxs[2] // == length of the leading "#"s
		rest := line[idxs[1]:]     // == text after the leading "#"s

		t := &Text{}
		if l := findText(rest); l != nil {
			t.writeLine(rest)
		}
		h := &Heading{
			Level:    level,
			Children: []InlineNode{t},
		}

		return h
	} else {
		return nil
	}
}

// tryOpenRaw tests if the line opens one of the two raw blocks (verbatim or
// macro). If it does, it returns the new node, along with the `delim` string
// that will close it. Otherwise, it returns nil and an empty string.
func tryOpenRaw(line []byte) (raw rawBlock, delim string) {
	if m := reRawOpener.FindSubmatch(line); m != nil {
		delim := string(m[1])
		tag := string(m[2])
		if len(tag) > 0 && tag[0] == '@' {
			// Tags that start with '=' indicate macros.
			return &Macro{Tag: tag[1:]}, delim
		} else {
			return &Verbatim{Lang: tag}, delim
		}
	} else {
		return nil, ""
	}
}

// tryOpenBlockquote tests if the line opens a blockquote. If it does, it
// returns the new node, along with the `delim` string that will close it.
func tryOpenBlockquote(line []byte) (blockquote *Blockquote, delim string) {
	if m := reBlockquoteOpener.FindSubmatch(line); m != nil {
		delim := string(m[1])
		return &Blockquote{}, delim
	}

	return nil, ""
}

// tryOpenList tests if the line opens a list. If it does, it returns the new
// node, and the offset of the list, which is the number of spaces that
// following lines must be indented to be included in the list.
func tryOpenList(line []byte) (list listBlock, offset int) {
	if idxs := reUnorderdListItemPrefix.FindIndex(line); idxs != nil {
		return &UnorderedList{}, idxs[1]
	} else if idxs := reOrderedListItemPrefix.FindIndex(line); idxs != nil {
		return &OrderedList{}, idxs[1]
	} else {
		return nil, 0
	}
}

// blockState represents a possible state that the [blockParser] can be in.
// The parser functions as a state machine: it pushes each line to the current
// state, which returns the next state. This process repeats until there are
// no more lines.
type blockState interface {
	pushLine(line []byte) (next blockState)
}

// paragraphState handles parsing within a paragraph.
type paragraphState struct {
	parent blockState
	text   lineWriter
}

// pushLine accepts any non-blank line. If it encounters a blank line,
// it returns the parent state.
func (p *paragraphState) pushLine(line []byte) (next blockState) {
	if t := findText(line); t != nil {
		p.text.writeLine(t)
		return p
	} else {
		return p.parent
	}
}

// listState handles parsing within an ordered or unordered list.
type listState struct {
	parent          blockState
	list            listItemAppender
	offset          int
	itemStartPrefix *regexp.Regexp
}

// pushLine implements the [blockState] interface. It accepts lines as long as
// they parse as line items for the current list type (ordered or unordered).
// Once a line doesn't match the current list, it defers to the parent state
// instead.
func (l *listState) pushLine(line []byte) blockState {
	if idxs := l.itemStartPrefix.FindIndex(line); idxs != nil {
		// Start a new list item.
		item := &ListItem{}
		l.list.appendListItems(item)

		return newListItemState(l, item, idxs[1], line)
	}

	// If we reach this point, the current line is not part of the list, so we
	// let the parent state handle it instead.
	return l.parent.pushLine(line)
}

// newListState creates a new [listState].
func newListState(parent blockState, list listBlock, offset int) *listState {
	var marker string
	switch l := list.(type) {
	case *OrderedList:
		marker = `\+`
	case *UnorderedList:
		marker = "-"
	default:
		panic(fmt.Sprintf("unrecognized list type: %T", l))
	}

	return &listState{
		parent:          parent,
		list:            list,
		offset:          offset,
		itemStartPrefix: regexp.MustCompile(fmt.Sprintf(`^ {%d}%s\s`, offset-2, marker)),
	}
}

// listItemState handles parsing within a list item.
type listItemState struct {
	parent blockState
	sub    blockState
	prefix *regexp.Regexp
}

// pushLine implements the [blockState] interface. It accepts lines that are
// indented at least as many spaces as the list item itself. If the line
// doesn't match, it pushes the line to its parent instead.
func (li *listItemState) pushLine(line []byte) blockState {
	if reBlankLine.Match(line) {
		li.sub = li.sub.pushLine(line)
		return li
	} else if idxs := li.prefix.FindIndex(line); idxs != nil {
		rest := line[idxs[1]:]
		li.sub = li.sub.pushLine(rest)
		return li
	} else {
		return li.parent.pushLine(line)
	}
}

// newListItemState returns a new [listItemState].
func newListItemState(
	parent blockState,
	item *ListItem,
	offset int,
	line []byte,
) *listItemState {
	c := &containerState{
		container: item,
	}

	return &listItemState{
		parent: parent,
		sub:    c.pushLine(line[offset:]),
		prefix: regexp.MustCompile(fmt.Sprintf(`^ {%d}`, offset)),
	}
}

// rawState handles parsing within a raw block (verbatim or macro).
type rawState struct {
	parent blockState
	node   lineWriter
	delim  *regexp.Regexp
}

// pushLine implements the [blockState] interface. It accepts lines until
// it reaches one that matches the delimiter pattern.
func (v *rawState) pushLine(line []byte) (next blockState) {
	if v.delim.Match(line) {
		return v.parent
	} else {
		v.node.writeLine(line)
		return v
	}
}

// containerState handles parsing in contexts that can contain multiple blocks,
// like blockquotes, multi-line list items, or the root document itself.
type containerState struct {
	container blockAppender
}

// pushLine implements the [blockState] interface. It tests each line for
// various patterns to see if it opens a new block. If it does, it adds the
// block as a child of the container, and may return a new state to continue
// that block, or return itself to move on to the next block.
func (c *containerState) pushLine(line []byte) blockState {
	if r, delim := tryOpenRaw(line); r != nil {
		c.container.appendBlocks(r)
		return &rawState{
			parent: c,
			node:   r,
			delim:  regexp.MustCompile(`^\s*` + delim + `\s*$`),
		}
	}

	if b, delim := tryOpenBlockquote(line); b != nil {
		c.container.appendBlocks(b)
		return newDelimitedContainerState(c, b, delim)
	}

	if h := tryOpenHeading(line); h != nil {
		c.container.appendBlocks(h)
		// Headings cannot be multi-line; return the current state.
		return c
	}

	if l, offset := tryOpenList(line); l != nil {
		c.container.appendBlocks(l)
		listState := newListState(c, l, offset)

		// Process the current line as part of the list.
		return listState.pushLine(line)
	}

	if p, t := tryOpenParagraph(line); p != nil {
		c.container.appendBlocks(p)
		return &paragraphState{
			parent: c,
			text:   t,
		}
	}

	// At this point, the line must be blank; return the current state to keep
	// seeking for the next block.
	return c
}

// delimitedContainerState handles parsing within a block that can contain
// multiple other blocks (see [containerState]) and stops parsing when it
// encounters a line that matches its delimiter pattern.
type delimitedContainerState struct {
	parent blockState
	sub    blockState
	delim  *regexp.Regexp
}

// pushLine implements the [blockState] interface. It functions like a
// [containerState] until it reaches a line that matches its delimiter
// pattern. Once it does, it returns to its parent state.
func (d *delimitedContainerState) pushLine(line []byte) blockState {
	if d.delim.Match(line) {
		return d.parent
	} else {
		d.sub = d.sub.pushLine(line)
		return d
	}
}

// newDelimitedContainerState returns a [delimitedContainerState] that adds
// blocks to the given `container` until it reaches a line that matches `delim`.
func newDelimitedContainerState(parent blockState, container blockAppender, delim string) *delimitedContainerState {
	return &delimitedContainerState{
		parent: parent,
		sub:    &containerState{container: container},
		delim:  regexp.MustCompile(`^\s*` + delim + `\s*$`),
	}
}

// blockParser parses the block structure of a KWML source document.
type blockParser struct {
	scanner  *bufio.Scanner
	document *Document
	current  blockState
}

// parse scans lines from the internal scanner to build up a tree of blocks.
// It returns an error if the scanner encounters one. Otherwise, it returns
// the parsed document.
func (b *blockParser) parse() (doc *Document, err error) {
	for b.scanner.Scan() {
		if err := b.scanner.Err(); err != nil {
			return nil, err
		}

		line := b.scanner.Bytes()
		b.current = b.current.pushLine(line)
	}

	return b.document, nil
}

// newBlockParser returns a new [blockParser] that will parse from the given
// [io.Reader].
func newBlockParser(r io.Reader) *blockParser {
	doc := &Document{}
	return &blockParser{
		scanner:  bufio.NewScanner(r),
		document: doc,
		current:  &containerState{doc},
	}
}
