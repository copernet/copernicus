package mem

import (
	"bytes"
	"math/rand"
)

type Immutable struct {
	root      *Node
	count     int
	totalSize uint64
}

func NewImmutable() *Immutable {
	return &Immutable{}
}

func newImmutable(root *Node, count int, totalSize uint64) *Immutable {
	table := &Immutable{
		root:      root,
		count:     count,
		totalSize: totalSize,
	}
	return table
}

func (m *Immutable) Len() int {
	return m.count
}

func (m *Immutable) Size() uint64 {
	return m.totalSize
}

func (m *Immutable) Has(key []byte) bool {
	if node := m.Get(key); node != nil {
		return true
	}
	return false
}

func (m *Immutable) Get(key []byte) []byte {
	for node := m.root; node != nil; {
		// Traverse left or right depending on the result of the
		// comparison.
		compareResult := bytes.Compare(key, node.key)
		if compareResult < 0 {
			node = node.left
			continue
		}
		if compareResult > 0 {
			node = node.right
			continue
		}
		return node.value
	}
	// A nil node was reached which means the key does not exist.
	return nil
}

func (m *Immutable) Put(key, value []byte) *Immutable {
	// Use an empty byte slice for the value when none was provided.  This
	// ultimately allows key existence to be determined from the value since
	// an empty byte slice is distinguishable from nil.
	if value == nil {
		value = emptySlice
	}
	// The node is the root of the tree if there isn't already one.
	if m.root == nil {
		root := NewNode(key, value, rand.Int())
		return newImmutable(root, 1, root.Size())
	}
	// Find the binary tree insertion point and construct a replaced list of
	// parents while doing so.  This is done because this is an immutable
	// data structure so regardless of where in the treap the new key/value
	// pair ends up, all ancestors up to and including the root need to be
	// replaced.
	//
	// When the key matches an entry already in the treap, replace the node
	// with a new one that has the new value set and return.
	var parents Stack
	var compareResult int
	for node := m.root; node != nil; {
		// Clone the node and link its parent to it if needed.
		nodeCopy := node.CLone()
		if oldParent := parents.At(0); oldParent != nil {
			if oldParent.left == node {
				oldParent.left = nodeCopy
			} else {
				oldParent.right = nodeCopy
			}
		}
		parents.Push(nodeCopy)
		// Traverse left or right depending on the result of comparing
		// the keys.
		compareResult = bytes.Compare(key, node.key)
		if compareResult < 0 {
			node = node.left
			continue
		}
		if compareResult > 0 {
			node = node.right
			continue
		}
		// The key already exists, so update its value.
		nodeCopy.value = value
		// Return new immutable treap with the replaced node and
		// ancestors up to and including the root of the tree.
		newRoot := parents.At(parents.Len() - 1)
		newTotalSize := m.totalSize - uint64(len(node.value)) +
			uint64(len(value))
		return newImmutable(newRoot, m.count, newTotalSize)
	}
	// Link the new node into the binary tree in the correct position.
	node := NewNode(key, value, rand.Int())
	parent := parents.At(0)
	if compareResult < 0 {
		parent.left = node
	} else {
		parent.right = node
	}
	// Perform any rotations needed to maintain the min-heap and replace
	// the ancestors up to and including the tree root.
	newRoot := parents.At(parents.Len() - 1)
	for parents.Len() > 0 {
		// There is nothing left to do when the node's priority is
		// greater than or equal to its parent's priority.
		parent = parents.Pop()
		if node.priority >= parent.priority {
			break
		}
		// Perform a right rotation if the node is on the left side or
		// a left rotation if the node is on the right side.
		if parent.left == node {
			node.right, parent.left = parent, node.right
		} else {
			node.left, parent.right = parent, node.left
		}
		// Either set the new root of the tree when there is no
		// grandparent or relink the grandparent to the node based on
		// which side the old parent the node is replacing was on.
		grandparent := parents.At(0)
		if grandparent == nil {
			newRoot = node
		} else if grandparent.left == parent {
			grandparent.left = node
		} else {
			grandparent.right = node
		}
	}
	return newImmutable(newRoot, m.count+1, t.totalSize+nodeSize(node))
}

func (t *Immutable) Delete(key []byte) *Immutable {
	// Find the node for the key while constructing a list of parents while
	// doing so.
	var parents Stack
	var delNode *Node
	for node := m.root; node != nil; {
		parents.Push(node)
		// Traverse left or right depending on the result of the
		// comparison.
		compareResult := bytes.Compare(key, node.key)
		if compareResult < 0 {
			node = node.left
			continue
		}
		if compareResult > 0 {
			node = node.right
			continue
		}
		// The key exists.
		delNode = node
		break
	}
	// There is nothing to do if the key does not exist.
	if delNode == nil {
		return m
	}
	// When the only node in the tree is the root node and it is the one
	// being deleted, there is nothing else to do besides removing it.
	parent := parents.At(1)
	if parent == nil && delNode.left == nil && delNode.right == nil {
		return newImmutable(nil, 0, 0)
	}
	// Construct a replaced list of parents and the node to delete itself.
	// This is done because this is an immutable data structure and
	// therefore all ancestors of the node that will be deleted, up to and
	// including the root, need to be replaced.
	var newParents parentStack
	for i := parents.Len(); i > 0; i-- {
		node := parents.At(i - 1)
		nodeCopy := cloneTreapNode(node)
		if oldParent := newParents.At(0); oldParent != nil {
			if oldParent.left == node {
				oldParent.left = nodeCopy
			} else {
				oldParent.right = nodeCopy
			}
		}
		newParents.Push(nodeCopy)
	}
	delNode = newParents.Pop()
	parent = newParents.At(0)
	// Perform rotations to move the node to delete to a leaf position while
	// maintaining the min-heap while replacing the modified children.
	var child *treapNode
	newRoot := newParents.At(newParents.Len() - 1)
	for delNode.left != nil || delNode.right != nil {
		// Choose the child with the higher priority.
		var isLeft bool
		if delNode.left == nil {
			child = delNode.right
		} else if delNode.right == nil {
			child = delNode.left
			isLeft = true
		} else if delNode.left.priority >= delNode.right.priority {
			child = delNode.left
			isLeft = true
		} else {
			child = delNode.right
		}
		// Rotate left or right depending on which side the child node
		// is on.  This has the effect of moving the node to delete
		// towards the bottom of the tree while maintaining the
		// min-heap.
		child = cloneTreapNode(child)
		if isLeft {
			child.right, delNode.left = delNode, child.right
		} else {
			child.left, delNode.right = delNode, child.left
		}
		// Either set the new root of the tree when there is no
		// grandparent or relink the grandparent to the node based on
		// which side the old parent the node is replacing was on.
		//
		// Since the node to be deleted was just moved down a level, the
		// new grandparent is now the current parent and the new parent
		// is the current child.
		if parent == nil {
			newRoot = child
		} else if parent.left == delNode {
			parent.left = child
		} else {
			parent.right = child
		}
		// The parent for the node to delete is now what was previously
		// its child.
		parent = child
	}
	// Delete the node, which is now a leaf node, by disconnecting it from
	// its parent.
	if parent.right == delNode {
		parent.right = nil
	} else {
		parent.left = nil
	}
	return newImmutable(newRoot, m.count-1, m.totalSize-delNode.Size())
}

func (m *Immutable) ForEach(fn func(k, v []byte) bool) {
	// Add the root node and all children to the left of it to the list of
	// nodes to traverse and loop until they, and all of their child nodes,
	// have been traversed.
	var parents Stack
	for node := m.root; node != nil; node = node.left {
		parents.Push(node)
	}
	for parents.Len() > 0 {
		node := parents.Pop()
		if !fn(node.key, node.value) {
			return
		}
		// Extend the nodes to traverse by all children to the left of
		// the current node's right child.
		for node := node.right; node != nil; node = node.left {
			parents.Push(node)
		}
	}
}
