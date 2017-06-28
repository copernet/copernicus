package mem

const (
	STATIC_DEPTH     = 128
	NODE_FIELDS_SIZE = 72
)

var (
	emptySlice = make([]byte, 0)
)

//type Node struct {
//	key      []byte
//	value    []byte
//	priority int
//	left     *node
//	right    *node
//}

//func NewNode(key, value []byte, priority int) *Node {
//	node := &Node{
//		key:      key,
//		value:    value,
//		priority: priority,
//	}
//	return node
//}
//func (m *Node) Size() uint64 {
//	return NODE_FIELDS_SIZE + uint64(len(m.key)+len(node.value))
//}
//
//func (m *Node) CLone() *Node {
//	node := &Node{
//		key:      m.key,
//		value:    m.value,
//		priority: m.priority,
//		left:     m.left,
//		right:    m.right,
//	}
//	return node
//}
//
//type Stack struct {
//	index    int
//	items    [STATIC_DEPTH]*Node
//	overflow []*Node
//}
//
//func (m *Stack) Len() int {
//	return m.index
//}
//
//func (m *Stack) At(n int) *Node {
//	index := s.index - n - 1
//	if index < 0 {
//		return nil
//	}
//	if index < STATIC_DEPTH {
//		return s.items[index]
//	}
//	return s.overflow[index-STATIC_DEPTH]
//}
//
//func (m *Stack) Pop() *Node {
//	if s.index == 0 {
//		return nil
//	}
//	s.index--
//	if s.index < STATIC_DEPTH {
//		node = s.items[s.index]
//		s.items[s.index] = nil
//		return node
//	}
//	node := s.overflow[s.index-STATIC_DEPTH]
//	s.overflow[s.index-STATIC_DEPTH] = nil
//	return node
//}
//
//func (m *Stack) Push(node *Node) {
//	if m.index < STATIC_DEPTH {
//		m.items[s.index] = node
//		m.index++
//		return
//	}
//	index := m.index - STATIC_DEPTH
//	if index+1 > cap(m.overflow) {
//		overflow := make([]*Node, index+1)
//		copy(overflow, m.overflow)
//		m.overflow = overflow
//	}
//	m.overflow[index] = node
//	m.index++
//}
