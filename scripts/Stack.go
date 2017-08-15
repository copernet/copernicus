package scripts

type Stack struct {
	stk [][]byte
}

func (s *Stack) PushStack(bytes []byte) bool {
	s.stk = append(s.stk, bytes)
	return true
}

// PopStack return valuas:
// first []byte is the value which is popped, second []byte is the error information if failed
func (s *Stack) PopStack() ([]byte, []byte) {
	sz := int32(len(s.stk))
	if sz <= 0 {
		return nil, []byte("the stack is empty")
	}
	so := s.stk[sz-1]
	s.stk = s.stk[:sz-1]

	return so, nil
}
func (s *Stack) PushBack(data []byte) {

}

func (s *Stack) Last() []byte {
	return s.stk[len(s.stk)-1]
}

func (s *Stack) Empty() bool {
	return len(s.stk) == 0
}

func (s *Stack) Size() int {
	return len(s.stk)
}
func (s *Stack) StackTop(i int) []byte {
	return s.stk[s.Size()+i]
}

func Swap(stack *Stack, other *Stack) {
	for i := 0; i < len(stack.stk); i++ {
		stack.stk[i] = other.stk[i]
	}
}

func NewStack() *Stack {
	stack := Stack{}
	return &stack
}
