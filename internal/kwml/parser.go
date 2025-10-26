package kwml

import (
	"io"
)

func Parse(r io.Reader) (doc *Document, err error) {
	b := newBlockParser(r)
	doc, err = b.parse()
	if err != nil {
		return nil, err
	}

	i := &inlineVisitor{}
	Walk(doc, i)

	n := &normalizer{}
	Walk(doc, n)

	return doc, nil
}
