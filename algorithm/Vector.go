package algorithm

import (
	"github.com/pkg/errors"
)

type Vector struct {
	array []interface{}
}

func (v *Vector) PushBack(value interface{}) {
	v.array = append(v.array, value)
}

func (v *Vector) PopBack() (interface{}, error) {
	stackLen := len(v.array)
	if stackLen == 0 {
		return nil, errors.New("stack is empty")
	}
	e := v.array[stackLen-1]
	v.array = v.array[:stackLen-1]
	if e != nil {
		return e, nil
	}
	return nil, errors.New("value is nil")
}

func (v *Vector) Begin() int {
	return 0
}

func (v *Vector) End() int {
	return v.Size() - 1
}

func (v *Vector) Front() interface{} {

	return v.array[0]
}

func (v *Vector) Back() interface{} {
	if v.Size() == 0 {
		return nil
	}
	return v.array[v.Size()-1]
}

func (v *Vector) At(index int) (interface{}, error) {
	if index > v.Size()-1 || index < 0 {
		return nil, errors.Errorf("vector index(%d) is error", index)
	}
	return v.array[index], nil
}

func (v *Vector) RemoveAt(index int) error {
	if index > v.Size()-1 || index < 0 {
		return errors.Errorf("vector index(%d) is error", index)
	}
	v.array = append(v.array[:index], v.array[index+1:]...)
	return nil

}

func (v *Vector) Clear() {
	v.array = make([]interface{}, 0)

}

func (v *Vector) Size() int {
	return len(v.array)
}

func (v *Vector) Empty() bool {
	return v.Size() == 0
}

func (v *Vector) Equal(other *Vector) bool {
	if v.Size() != other.Size() {
		return false
	}
	for i := 0; i < v.Size(); i++ {
		if v.array[i] != other.array[i] {
			return false
		}
	}
	return true
}

func SwapVector(v *Vector, other *Vector) {
	if v.Size() == 0 && other.Size() == 0 {
		return
	}
	if other.Size() == 0 {
		other.array = v.array[:v.Size()]
		v.array = make([]interface{}, 0)
	}
	if v.Size() == 0 {
		v.array = other.array[:other.Size()]
		other.array = make([]interface{}, 0)
	}

	v.array, other.array = other.array, v.array
}

func NewVector() *Vector {
	array := make([]interface{}, 0)
	return &Vector{array}
}
