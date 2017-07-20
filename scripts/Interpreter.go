package scripts

type Interpreter struct {
	stack *Stack
}

func (interpreter *Interpreter) Verify() (bool) {
	return false
}
func (interpreter *Interpreter) Exec() (bool) {
	return false
}
