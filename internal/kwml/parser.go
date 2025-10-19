package kwml

import (
	"io"
)

func Parse(r io.Reader) (doc *Document, err error) {
	b := NewBlockParser(r)
	return b.Parse()
}
