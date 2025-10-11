package kwml

import (
	"bufio"
	"io"
	"regexp"
	"slices"

	"github.com/cmessinides/kiwi/internal/ast"
)

type Block int

const (
	BlockUnknown Block = iota
	BlockDocument
	BlockHeading
	BlockParagraph
	BlockVerbatim
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
	default:
		return "unknown"
	}
}

var (
	reNonBlankLine  = regexp.MustCompile(`^\s*(\S|\S.*\S)\s*$`)
	reHeadingPrefix = regexp.MustCompile(`^\s*(#{1,6})\s`)
	// /x60 = literal backtick
	reVerbatimOpener = regexp.MustCompile(`^\s*(\x60{3,})(\w*)\s*$`)
)

func findText(line []byte) []byte {
	match := reNonBlankLine.FindSubmatch(line)
	if match != nil {
		// The first submatch is just the text, with leading and trailing
		// whitespace trimmed.
		return match[1]
	} else {
		return nil
	}
}

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

func newParagraph(text []byte) *ast.Node[Block] {
	p := ast.NewNode(BlockParagraph)
	appendLine(p, text)
	return p
}

func newHeading(level int, rest []byte) *ast.Node[Block] {
	h := ast.NewNode(BlockHeading).SetAttr("level", level)
	if t := findText(rest); t != nil {
		appendLine(h, t)
	}

	return h
}

func tryOpenVerbatim(line []byte) (verbatim *ast.Node[Block], delim string) {
	if m := reVerbatimOpener.FindSubmatch(line); m != nil {
		delim := string(m[1])
		lang := string(m[2])
		return ast.NewNode(BlockVerbatim).SetAttr("lang", lang), delim
	} else {
		return nil, ""
	}
}

type blockState interface {
	pushLine(line []byte) (next blockState)
}

type paragraphState struct {
	parent blockState
	para   *ast.Node[Block]
}

func (p *paragraphState) pushLine(line []byte) (next blockState) {
	if t := findText(line); t != nil {
		appendLine(p.para, t)
		return p
	} else {
		return p.parent
	}
}

type verbatimState struct {
	parent blockState
	node   *ast.Node[Block]
	delim  *regexp.Regexp
}

func (v *verbatimState) pushLine(line []byte) (next blockState) {
	if v.delim.Match(line) {
		return v.parent
	} else {
		appendLine(v.node, line)
		return v
	}
}

type documentState struct {
	doc *ast.Node[Block]
}

func (d *documentState) pushLine(line []byte) blockState {
	if v, delim := tryOpenVerbatim(line); v != nil {
		d.doc.Append(v)

		return &verbatimState{
			parent: d,
			node:   v,
			delim:  regexp.MustCompile(`^\s*` + delim + `\s*$`),
		}
	}

	if idxs := reHeadingPrefix.FindSubmatchIndex(line); idxs != nil {
		level := idxs[3] - idxs[2]
		rest := line[idxs[1]:]

		d.doc.Append(
			newHeading(level, rest),
		)
		return d
	}

	if t := findText(line); t != nil {
		p := newParagraph(t)
		d.doc.Append(p)
		return &paragraphState{
			parent: d,
			para:   p,
		}
	} else {
		return d
	}
}

type BlockParser struct {
	scanner  *bufio.Scanner
	document *ast.Node[Block]
	current  blockState
}

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

func NewBlockParser(r io.Reader) *BlockParser {
	doc := ast.NewNode(BlockDocument)
	return &BlockParser{
		scanner:  bufio.NewScanner(r),
		document: doc,
		current:  &documentState{doc},
	}
}
