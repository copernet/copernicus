package scripts

type Interpreter struct {
	stack *Stack
}

func (interpreter *Interpreter) Verify() bool {
	return false
}

func (interpreter *Interpreter) Exec(script *Script) bool {
	return false
}

func NewInterpreter() *Interpreter {
	return nil
}
