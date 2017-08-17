package algorithm

import (
	"testing"
)

func TestVector(t *testing.T) {
	vector := NewVector()
	vector.PushBack(1)
	vector.PushBack(2)
	vector.PushBack(3)
	vector.PushBack(4)
	vector.PushBack(5)
	len := vector.Size()
	if len != 5 {
		t.Errorf("get vector size  failed , Got %d ,ecpected 5", len)
	}
	value, err := vector.PopBack()
	if err != nil {
		t.Errorf("PopStack error, %s", err.Error())
	}
	if value.(int) != 5 {
		t.Errorf("PopStack failed ,Got %d ,expected 5", value)
	}
	len = vector.Size()
	if len != 4 {
		t.Errorf("get vector size  failed , Got %d ,ecpected 4", len)
	}

	value, err = vector.PopBack()
	if err != nil {
		t.Errorf("PopBack error, %s", err.Error())
	}
	if value.(int) != 4 {
		t.Errorf("PopBack failed ,Got %d ,expected 4", value)
	}
	empty := vector.Empty()
	if empty {
		t.Errorf("vector is not empty . Got %v, expected false.", empty)
	}
	vector.PopBack()
	vector.PopBack()
	vector.PopBack()
	value, err = vector.PopBack()
	if err == nil {
		t.Errorf("we should get error")
	}
	if value != nil {
		t.Errorf("PopBack failed ,Got %d ,expected 4", value)
	}
}

func TestSwapVector(t *testing.T) {
	vector := NewVector()
	vector.PushBack(1)
	vector.PushBack(2)

	vectorTest := NewVector()
	vectorTest.PushBack(1)
	vectorTest.PushBack(2)

	vectorOther := NewVector()
	vectorOther.PushBack(3)
	vectorOther.PushBack(4)

	vectorOtherTest := NewVector()
	vectorOtherTest.PushBack(3)
	vectorOtherTest.PushBack(4)

	SwapVector(vector, vectorOther)
	if !vector.Equal(vectorOtherTest) || !vectorOther.Equal(vectorTest) {
		t.Errorf("swap vector failed")
	}
	vectorOther.PopBack()
	vectorOther.PopBack()

	SwapVector(vector, vectorOther)
	if !vector.Empty() || !vectorOther.Equal(vectorOtherTest) {
		t.Errorf("swap empty stacl failed")
	}

}
