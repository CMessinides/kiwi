// Package ast provides generic data structures abstract syntax trees (ASTs),
// as well as functions for working with them.
package ast

import "slices"

// Node represents a node in an AST.
type Node[T comparable] struct {
	Type     T
	parent   *Node[T]
	children []*Node[T]
	attrs    *AttributeMap
}

// Parent returns the node's parent, if it has one.
func (n *Node[T]) Parent() *Node[T] {
	return n.parent
}

// Children returns a slice of the node's children.
func (n *Node[T]) Children() []*Node[T] {
	return n.children
}

// FirstChild returns the node's first child, or nil if it has no children.
func (n *Node[T]) FirstChild() *Node[T] {
	if len(n.children) == 0 {
		return nil
	}

	return n.children[0]
}

// LastChild returns the node's last child, or nil if it has no children.
func (n *Node[T]) LastChild() *Node[T] {
	if len(n.children) == 0 {
		return nil
	}

	return n.children[len(n.children)-1]
}

// Append adds the given `nodes` as children to this node and
// returns it.
func (n *Node[T]) Append(nodes ...*Node[T]) *Node[T] {
	if len(nodes) == 0 {
		return n
	}

	added := make([]*Node[T], 0, len(nodes))
	for _, node := range nodes {
		if !slices.Contains(n.children, node) {
			node.parent = n
			added = append(added, node)
		}
	}

	if len(added) == 0 {
		return n
	}

	n.children = append(n.children, added...)
	return n
}

// Remove removes the given `nodes` from this node and returns it.
func (n *Node[T]) Remove(nodes ...*Node[T]) *Node[T] {
	if len(nodes) == 0 {
		return n
	}

	keep := make([]*Node[T], 0, len(n.children))
	for _, child := range n.children {
		if slices.Contains(nodes, child) {
			child.parent = nil
		} else {
			keep = append(keep, child)
		}
	}

	n.children = keep
	return n
}

// Attrs returns the node's attributes.
func (n *Node[T]) Attrs() *AttributeMap {
	return n.attrs
}

// Attr returns the value of the attribute associated with `key`. The second
// return value will be false if the attribute does not exist.
func (n *Node[T]) Attr(key string) (value any, ok bool) {
	return n.attrs.Get(key)
}

// HasAttr returns whether the node has attribute `key`.
func (n *Node[T]) HasAttr(key string) bool {
	return n.attrs.Has(key)
}

// SetAttr sets the attribute associated with `key` to `value`.
func (n *Node[T]) SetAttr(key string, value any) *Node[T] {
	n.attrs.Set(key, value)
	return n
}

// RemoveAttr removes the attribute associated with `key`.
func (n *Node[T]) RemoveAttr(key string) bool {
	return n.attrs.Delete(key)
}

// NewNode constructs a new node of type `t`.
func NewNode[T comparable](t T) *Node[T] {
	return &Node[T]{Type: t, attrs: NewAttributeMap()}
}

// NodeVisitor is the interface that wraps the VisitNode method. It can visit
// nodes recursively by calling `VisitNode` on the children of each node it
// visits.
type NodeVisitor[T comparable] interface {
	VisitNode(node *Node[T]) error
}

// NodeVisitorFunc is a function that implements the [NodeVisitorFunc] interface.
type NodeVisitorFunc[T comparable] func(node *Node[T]) error

// VisitNode implements the [NodeVisitor] interface by calling itself on
// the node.
func (f NodeVisitorFunc[T]) VisitNode(node *Node[T]) error {
	return f(node)
}
