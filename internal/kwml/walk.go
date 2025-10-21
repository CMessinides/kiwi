package kwml

import "iter"

type Visitor interface {
	StartVisit(node Node) (v Visitor)
	EndVisit(node Node)
}

type childrenIterator interface {
	iterChildren() iter.Seq[Node]
}

func blockIter(blocks []BlockNode) iter.Seq[Node] {
	return func(yield func(Node) bool) {
		for _, b := range blocks {
			if !yield(b) {
				return
			}
		}
	}
}

func inlineIter(inlines []InlineNode) iter.Seq[Node] {
	return func(yield func(Node) bool) {
		for _, i := range inlines {
			if !yield(i) {
				return
			}
		}
	}
}

func listItemIter(items []*ListItem) iter.Seq[Node] {
	return func(yield func(Node) bool) {
		for _, i := range items {
			if !yield(i) {
				return
			}
		}
	}
}

func Walk(node Node, v Visitor) {
	if v = v.StartVisit(node); v == nil {
		return
	}

	if i, ok := node.(childrenIterator); ok {
		for c := range i.iterChildren() {
			Walk(c, v)
		}
	}

	v.EndVisit(node)
}
