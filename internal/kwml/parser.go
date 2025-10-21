package kwml

import (
	"io"
)

func Parse(r io.Reader) (doc *Document, err error) {
	b := newBlockParser(r)
	return b.parse()
}
