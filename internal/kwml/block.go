package kwml

import (
	"bufio"
	"io"
	"regexp"
	"slices"

	"github.com/cmessinides/kiwi/internal/ast"
)

// Block represents the type of a KWML block.
type Block int

const (
	BlockUnknown Block = iota
	BlockDocument
	BlockHeading
	BlockParagraph
	BlockVerbatim
	BlockMacro
)

// String implements the [fmt.Stringer] interface.
func (b Block) String() string {
	switch b {
	case BlockDocument:
		return "document"
	case BlockHeading:
		return "heading"
	case BlockParagraph:
		return "paragraph"
	case BlockVerbatim:
		return "verbatim"
	case BlockMacro:
		return "macro"
	default:
		return "unknown"
	}
}

var (
	reNonBlankLine  = regexp.MustCompile(`^\s*(\S|\S.*\S)\s*$`)
	reHeadingPrefix = regexp.MustCompile(`^\s*(#{1,6})\s`)
	// /x60 = literal backtick
	reRawOpener = regexp.MustCompile(`^\s*(\x60{3,})(@\w+|\w*)\s*$`)
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

// appendLine joins the given `line` to the block's `"content"` attribute with
// a newline character (`\n`). If the block has no existing content, appendLine
// copies the data from the given slice into a new one, and sets the block's
// content to the new slice.
func appendLine(block *ast.Node[Block], line []byte) {
	existing, _ := block.Attr("content")

	if b, ok := existing.([]byte); ok {
		c := append(b, '\n')
		c = append(c, line...)

		block.SetAttr("content", c)
	} else {
		block.SetAttr("content", slices.Clone(line))
	}
}

// tryOpenParagraph tests if the line opens a paragraph. If it does, it returns
// the new node.
func tryOpenParagraph(line []byte) *ast.Node[Block] {
	if t := findText(line); t != nil {
		p := ast.NewNode(BlockParagraph)
		appendLine(p, t)
		return p
	}

	return nil
}

// tryOpenHeading tests if the line opens a heading. If it does, it returns the
// new node.
func tryOpenHeading(line []byte) *ast.Node[Block] {
	if idxs := reHeadingPrefix.FindSubmatchIndex(line); idxs != nil {
		level := idxs[3] - idxs[2] // == length of the leading "#"s
		rest := line[idxs[1]:]     // == text after the leading "#"s

		h := ast.NewNode(BlockHeading).SetAttr("level", level)
		if t := findText(rest); t != nil {
			appendLine(h, t)
		}

		return h
	} else {
		return nil
	}
}

// tryOpenRaw tests if the line opens one of the two raw blocks (verbatim or
// macro). If it does, it returns the new node, along with the `delim` string
// that will close it. Otherwise, it returns nil and an empty string.
func tryOpenRaw(line []byte) (verbatim *ast.Node[Block], delim string) {
	if m := reRawOpener.FindSubmatch(line); m != nil {
		delim := string(m[1])
		tag := string(m[2])
		if len(tag) > 0 && tag[0] == '@' {
			// Tags that start with '=' indicate macros.
			return ast.NewNode(BlockMacro).SetAttr("tag", tag[1:]), delim
		} else {
			return ast.NewNode(BlockVerbatim).SetAttr("lang", tag), delim
		}
	} else {
		return nil, ""
	}
}

// blockState represents a possible state that the [BlockParser] can be in.
// The parser functions as a state machine: it pushes each line to the current
// state, which returns the next state. This process repeats until there are
// no more lines.
type blockState interface {
	pushLine(line []byte) (next blockState)
}

// paragraphState handles parsing within a paragraph.
type paragraphState struct {
	parent blockState
	para   *ast.Node[Block]
}

// pushLine accepts any non-blank line. If it encounters a blank line,
// it returns the parent state.
func (p *paragraphState) pushLine(line []byte) (next blockState) {
	if t := findText(line); t != nil {
		appendLine(p.para, t)
		return p
	} else {
		return p.parent
	}
}

// rawState handles parsing within a raw block (verbatim or macro).
type rawState struct {
	parent blockState
	node   *ast.Node[Block]
	delim  *regexp.Regexp
}

// pushLine implements the [blockState] interface. It accepts lines until
// it reaches one that matches the delimiter pattern.
func (v *rawState) pushLine(line []byte) (next blockState) {
	if v.delim.Match(line) {
		return v.parent
	} else {
		appendLine(v.node, line)
		return v
	}
}

// documentState handles parsing when outside of any block.
type documentState struct {
	doc *ast.Node[Block]
}

// pushLine implements the [blockState] interface. It tests each line for
// various patterns to see if it opens a new block. If it doe, it adds the
// block as a child of the document, and may return a new state to continue
// that block, or return itself to move on to the next block.
func (d *documentState) pushLine(line []byte) blockState {
	if r, delim := tryOpenRaw(line); r != nil {
		d.doc.Append(r)
		return &rawState{
			parent: d,
			node:   r,
			delim:  regexp.MustCompile(`^\s*` + delim + `\s*$`),
		}
	}

	if h := tryOpenHeading(line); h != nil {
		d.doc.Append(h)
		// Headings cannot be multi-line; return the current state.
		return d
	}

	if p := tryOpenParagraph(line); p != nil {
		d.doc.Append(p)
		return &paragraphState{
			parent: d,
			para:   p,
		}
	}

	// At this point, the line must be blank; return the current state to keep
	// seeking for the next block.
	return d
}

// BlockParser parses the block structure of a KWML source document.
type BlockParser struct {
	scanner  *bufio.Scanner
	document *ast.Node[Block]
	current  blockState
}

// Parse scans lines from the internal scanner to build up a tree of blocks.
// It returns an error if the scanner encounters one. Otherwise, it returns
// the parsed document.
func (b *BlockParser) Parse() (doc *ast.Node[Block], err error) {
	for b.scanner.Scan() {
		if err := b.scanner.Err(); err != nil {
			return nil, err
		}

		line := b.scanner.Bytes()
		b.current = b.current.pushLine(line)
	}

	return b.document, nil
}

// NewBlockParser returns a new [BlockParser] that will parse from the given
// [io.Reader].
func NewBlockParser(r io.Reader) *BlockParser {
	doc := ast.NewNode(BlockDocument)
	return &BlockParser{
		scanner:  bufio.NewScanner(r),
		document: doc,
		current:  &documentState{doc},
	}
}
