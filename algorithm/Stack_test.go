package algorithm

import "testing"

func TestStack(t *testing.T) {

	stack := NewStack()
	stack.PushStack(1)
	stack.PushStack(2)
	stack.PushStack(3)
	stack.PushStack(4)
	stack.PushStack(5)
	len := stack.Size()
	if len != 5 {
		t.Errorf("get stack size  failed , Got %d ,ecpected 5", len)
	}
	value, err := stack.PopStack()
	if err != nil {
		t.Errorf("PopStack error, %s", err.Error())
	}
	if value.(int) != 5 {
		t.Errorf("PopStack failed ,Got %d ,expected 5", value)
	}
	len = stack.Size()
	if len != 4 {
		t.Errorf("get stack size  failed , Got %d ,ecpected 4", len)
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
