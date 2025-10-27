package kwml

import (
	"io"
)

func parseInlines(doc *Document) {
	Walk(doc, &inlineVisitor{})
}

func normalize(doc *Document) {
	Walk(doc, &normalizer{})
}

func Parse(r io.Reader) (doc *Document, err error) {
	b := newBlockParser(r)
	doc, err = b.parse()
	if err != nil {
		return nil, err
	}

	parseInlines(doc)
	normalize(doc)

	return doc, nil
}
