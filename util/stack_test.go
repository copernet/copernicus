package util

import (
	"reflect"
	"testing"
)

func TestStackCountBool(t *testing.T) {
	stack := NewStack()
	stack.Push(1)
	stack.Push(2)
	n := stack.CountBool(false)
	if n > 0 {
		t.Errorf("Want %d, got %d", 0, n)
	}

	stack.Push(false)
	n = stack.CountBool(false)
	if n != 1 {
		t.Errorf("Want %d, got %d", 1, n)
	}

	stack.Push(true)
	n = stack.CountBool(true)
	if n != 1 {
		t.Errorf("Want %d, got %d", 1, n)
	}
}

func TestStack(t *testing.T) {
	stack := NewStack()
	stack.Push(1)
	stack.Push(2)
	stack.Push(3)
	stack.Push(4)
	stack.Push(5)
	stack.Push(nil)
	length := stack.Size()
	if length != 6 {
		t.Errorf("get stack size  failed , Got %d ,ecpected 5", length)
		return
	}

	tmpStack := stack.Copy()
	if !stack.Equal(tmpStack) {
		t.Errorf("the stack should equal.")
	}

	value := stack.Pop()
	//var nilValue interface{}
	if reflect.TypeOf(value) != nil {
		t.Errorf("PopStack failed ,Got %d ,expected 5", value)
		return
	}
	value = stack.Pop()
	if value.(int) != 5 {
		t.Errorf("PopStack failed ,Got %d ,expected 5", value)
	}
	length = stack.Size()
	if length != 4 {
		t.Errorf("get stack size  failed , Got %d ,ecpected 4", length)
	}

	value = stack.Pop()
	if value.(int) != 4 {
		t.Errorf("PopStack failed ,Got %d ,expected 4", value)
	}

	empty := stack.Empty()
	if empty {
		t.Errorf("stack is not empty . Got %v, expected false.", empty)
	}
	stack.Pop()
	stack.Pop()
	stack.Pop()
	value = stack.Pop()
	if value != nil {
		t.Errorf("PopStack failed ,Got %d ,expected 4", value)
	}

	stack.Push(11)
	stack.Push(22)

	if ok := stack.Insert(1, 221); ok {
		v := stack.array[1]
		if v != 221 {
			t.Errorf("insert failed, v is:%d", v)
		}
	}

	if ok := stack.Swap(0, 1); ok {
		if stack.array[0] != 221 && stack.array[1] != 11 {
			t.Errorf("swap stack failed.")
		}
	}

	if ok := stack.Erase(0, 3); ok {
		if stack.Size() != 0 {
			t.Errorf("erase stack failed.")
		}
	}

	stack.Push(111)
	stack.Push(222)
	stack.Push(333)

	if ok := stack.RemoveAt(2); ok {
		if stack.Size() != 2 && stack.array[2] != 333 {
			t.Errorf("removeAt stack failed.")
		}
	}

	v := stack.Top(-1)
	if v != 222 {
		t.Errorf("top stack failed, the v is:%d", v)
	}

	if ok := stack.SetTop(-1, 333); ok {
		if stack.array[1] != 333 {
			t.Errorf("stack.array[2]:%d", stack.array[2])
		}
	}
}

func TestSwapStack(t *testing.T) {
	stack := NewStack()
	stack.Push(1)
	stack.Push(2)

	stackTest := NewStack()
	stackTest.Push(1)
	stackTest.Push(2)

	stackOther := NewStack()
	stackOther.Push(3)
	stackOther.Push(4)

	stackOtherTest := NewStack()
	stackOtherTest.Push(3)
	stackOtherTest.Push(4)

	Swap(stack, stackOther)

	if !stack.Equal(stackOtherTest) || !stackOther.Equal(stackTest) {
		t.Errorf("swap stack failed")
	}
	stackOther.Pop()
	stackOther.Pop()

	Swap(stack, stackOther)
	if !stack.Empty() || !stackOther.Equal(stackOtherTest) {
		t.Errorf("swap empty stacl failed")
	}

}
