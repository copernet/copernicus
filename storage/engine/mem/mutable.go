package mem

import (
	"bytes"
	"math/rand"
)

type Mutable struct {
	root      *Node
	count     int
	totalSize uint64
}

func NewMutable() *Mutable {
	return &Mutable{}
}

func (m *Mutable) Len() int {
	return m.count
}

func (m *Mutable) Size() uint64 {
	return m.totalSize
}

func (m *Mutable) get(key []byte) (*Node, *Node) {
	var parent *Node
	for node := m.root; node != nil; {
		compareResult := bytes.Compare(key, node.key)
		if compareResult < 0 {
			parent = node
			node = node.left
			continue
		}
		if compareResult > 0 {
			parent = node
			node = node.right
			continue
		}
		return node, parent
	}
	return nil, nil
}

func (m *Mutable) Has(key []byte) bool {
	if node, _ := m.get(key); node != nil {
		return true
	}
	return false
}

func (m *Mutable) Get(key []byte) []byte {
	if node, _ := m.get(key); node != nil {
		return node.value
	}
	return nil
}

func (m *Mutable) relinkGrandparent(node, parent, grandparent *Node) {
	if grandparent == nil {
		m.root = node
		return
	}
	if grandparent.left == parent {
		grandparent.left = node
	} else {
		grandparent.right = node
	}
}

func (m *Mutable) Put(key, value []byte) {
	if value == nil {
		value = emptySlice
	}
	if m.root == nil {
		node := NewNode(key, value, rand.Int())
		m.count = 1
		m.totalSize = node.Size()
		m.root = node
		return
	}
	var parents parentStack
	var compareResult int
	for node := m.root; node != nil; {
		parents.Push(node)
		compareResult = bytes.Compare(key, node.key)
		if compareResult < 0 {
			node = node.left
			continue
		}
		if compareResult > 0 {
			node = node.right
			continue
		}
		m.totalSize -= uint64(len(node.value))
		m.totalSize += uint64(len(value))
		node.value = value
		return
	}
	node := NewNode(key, value, rand.Int())
	m.count++
	m.totalSize += node.Size()
	parent := parents.At(0)
	if compareResult < 0 {
		parent.left = node
	} else {
		parent.right = node
	}
	for parents.Len() > 0 {
		parent = parents.Pop()
		if node.priority >= parent.priority {
			break
		}
		if parent.left == node {
			node.right, parent.left = parent, node.right
		} else {
			node.left, parent.right = parent, node.left
		}
		m.relinkGrandparent(node, parent, parents.At(0))
	}
}

func (m *Mutable) Delete(key []byte) {
	node, parent := m.get(key)
	if node == nil {
		return
	}
	if parent == nil && node.left == nil && node.right == nil {
		t.root = nil
		t.count = 0
		t.totalSize = 0
		return
	}
	var isLeft bool
	var child *Node
	for node.left != nil || node.right != nil {
		// Choose the child with the higher priority.
		if node.left == nil {
			child = node.right
			isLeft = false
		} else if node.right == nil {
			child = node.left
			isLeft = true
		} else if node.left.priority >= node.right.priority {
			child = node.left
			isLeft = true
		} else {
			child = node.right
			isLeft = false
		}
		if isLeft {
			child.right, node.left = node, child.right
		} else {
			child.left, node.right = node, child.left
		}
		m.relinkGrandparent(child, node, parent)
		parent = child
	}
	if parent.right == node {
		parent.right = nil
	} else {
		parent.left = nil
	}
	m.count--
	m.totalSize -= node.Size()
}

func (m *Mutable) ForEach(fn func(k, v []byte) bool) {
	var parents Stack
	for node := m.root; node != nil; node = node.left {
		parents.Push(node)
	}
	for parents.Len() > 0 {
		node := parents.Pop()
		if !fn(node.key, node.value) {
			return
		}
		for node := node.right; node != nil; node = node.left {
			parents.Push(node)
		}
	}
}

func (m *Mutable) Reset() {
	m.count = 0
	m.totalSize = 0
	m.root = nil
}
