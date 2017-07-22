package scripts

type Stack struct {
	vector [][]byte
}

func (stack *Stack) Push() bool {
	return false
}

func (stack *Stack) Pop() bool {
	return false
}

func NewStack() *Stack {
	return nil
}
