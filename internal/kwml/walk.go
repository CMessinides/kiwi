package kwml

type Visitor interface {
	StartVisit(node Node) (v Visitor)
	EndVisit(node Node)
}

func Walk(node Node, v Visitor) {
	if v = v.StartVisit(node); v == nil {
		return
	}

	if container, ok := node.(BlockContainer); ok {
		for b := range container.Blocks() {
			Walk(b, v)
		}
	} else if container, ok := node.(InlineContainer); ok {
		for i := range container.Inlines() {
			Walk(i, v)
		}
	}

	v.EndVisit(node)
}
