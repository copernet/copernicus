package algorithm

import (
	"testing"
)

func TestStack(t *testing.T) {

	stack := NewStack()
	stack.PushStack(1)
	stack.PushStack(2)
	stack.PushStack(3)
	stack.PushStack(4)
	stack.PushStack(5)
	length := stack.Size()
	if length != 5 {
		t.Errorf("get stack size  failed , Got %d ,ecpected 5", length)
	}
	value, err := stack.PopStack()
	if err != nil {
		t.Errorf("PopStack error, %s", err.Error())
	}
	if value.(int) != 5 {
		t.Errorf("PopStack failed ,Got %d ,expected 5", value)
	}
	length = stack.Size()
	if length != 4 {
		t.Errorf("get stack size  failed , Got %d ,ecpected 4", length)
	}

	value, err = stack.PopStack()
	if err != nil {
		t.Errorf("PopStack error, %s", err.Error())
	}
	if value.(int) != 4 {
		t.Errorf("PopStack failed ,Got %d ,expected 4", value)
	}
	empty := stack.Empty()
	if empty {
		t.Errorf("stack is not empty . Got %v, expected false.", empty)
	}
	stack.PopStack()
	stack.PopStack()
	stack.PopStack()
	value, err = stack.PopStack()
	if err == nil {
		t.Errorf("we should get error")
	}
	if value != nil {
		t.Errorf("PopStack failed ,Got %d ,expected 4", value)
	}

}

func TestSwapStack(t *testing.T) {
	stack := NewStack()
	stack.PushStack(1)
	stack.PushStack(2)

	stackTest := NewStack()
	stackTest.PushStack(1)
	stackTest.PushStack(2)

	stackOther := NewStack()
	stackOther.PushStack(3)
	stackOther.PushStack(4)

	stackOtherTest := NewStack()
	stackOtherTest.PushStack(3)
	stackOtherTest.PushStack(4)

	SwapStack(stack, stackOther)
	if !stack.Equal(stackOtherTest) || !stackOther.Equal(stackTest) {
		t.Errorf("swap stack failed")
	}
	stackOther.PopStack()
	stackOther.PopStack()

	SwapStack(stack, stackOther)
	if !stack.Empty() || !stackOther.Equal(stackOtherTest) {
		t.Errorf("swap empty stacl failed")
	}

}
