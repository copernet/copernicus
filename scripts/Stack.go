package scripts

type Stack struct {
	stk [][]byte
}

func (s *Stack) PushBytes(bytes []byte) bool {
	s.stk = append(s.stk, bytes)
	return true
}

// PopBytes return valuas:
// first []byte is the value which is popped, second []byte is the error information if failed
func (s *Stack) PopBytes() ([]byte, []byte) {
	sz := int32(len(s.stk))
	if sz <= 0 {
		return nil, []byte("the stack is empty")
	}
	so := s.stk[sz-1]
	s.stk = s.stk[:sz-1]

	return so, nil
}

func NewStack() *Stack {
	stack := Stack{}
	return &stack
}
