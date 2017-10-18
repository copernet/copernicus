package algorithm

import (
	"github.com/pkg/errors"
)

type Vector struct {
	Array []interface{}
}

func (v *Vector) PushBack(value interface{}) {
	v.Array = append(v.Array, value)
}

func (v *Vector) PopBack() (interface{}, error) {
	stackLen := len(v.Array)
	if stackLen == 0 {
		return nil, errors.New("stack is empty")
	}
	e := v.Array[stackLen-1]
	v.Array = v.Array[:stackLen-1]
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

	return v.Array[0]
}

func (v *Vector) Back() interface{} {
	if v.Size() == 0 {
		return nil
	}
	return v.Array[v.Size()-1]
}

func (v *Vector) SetBack(value interface{}) {
	v.Array[v.Size()-1] = value
}
func (v *Vector) At(index int) (interface{}, error) {
	if index > v.Size()-1 || index < 0 {
		return nil, errors.Errorf("vector index(%d) is error", index)
	}
	return v.Array[index], nil
}

func (v *Vector) RemoveAt(index int) error {
	if index > v.Size()-1 || index < 0 {
		return errors.Errorf("vector index(%d) is error", index)
	}
	v.Array = append(v.Array[:index], v.Array[index+1:]...)
	return nil

}

func (v *Vector) Clear() {
	v.Array = make([]interface{}, 0)

}

func (v *Vector) Size() int {
	return len(v.Array)
}

func (v *Vector) Empty() bool {
	return v.Size() == 0
}

func (v *Vector) Equal(other *Vector) bool {
	if v.Size() != other.Size() {
		return false
	}
	for i := 0; i < v.Size(); i++ {
		if v.Array[i] != other.Array[i] {
			return false
		}
	}
	return true
}

func (v *Vector) CountEqualElement(value bool) int {
	count := 0
	for i := 0; i < len(v.Array); i++ {
		if v.Array[i].(bool) == value {
			count++
		}
	}
	return count
}

func (v *Vector) ReverseArray() []interface{} {
	reverseArray := make([]interface{}, 0)
	for i := len(v.Array) - 1; i >= 0; i-- {
		reverseArray = append(reverseArray, v.Array[i])
	}
	v.Array = reverseArray
	return reverseArray

}

func (v *Vector) Copy() *Vector {
	newVec := NewVector()
	newVec.Array = append(newVec.Array, v.Array...)
	return newVec
}

func (v *Vector) Has(Item interface{}) bool {
	for _, val := range v.Array {
		if val == Item {
			return true
		}
	}
	return false
}

func (v *Vector) Each(f func(item interface{}) bool) {
	for _, v := range v.Array {
		if !f(v) {
			break
		}
	}
}

func SwapVector(v *Vector, other *Vector) {
	if v.Size() == 0 && other.Size() == 0 {
		return
	}
	if other.Size() == 0 {
		other.Array = v.Array[:v.Size()]
		v.Array = make([]interface{}, 0)
	}
	if v.Size() == 0 {
		v.Array = other.Array[:other.Size()]
		other.Array = make([]interface{}, 0)
	}

	v.Array, other.Array = other.Array, v.Array
}

func (v *Vector) SetValueByIndex(index int, value interface{}) error {
	if v.Size() <= index || index < 0 {
		return errors.Errorf("vector index (%d) is error, the vector have %d element", index, v.Size())
	}

	v.Array[index] = value
	return nil
}

func NewVectorWithSize(size uint) *Vector {
	array := make([]interface{}, size)
	return &Vector{array}
}

func NewVector() *Vector {
	array := make([]interface{}, 0)
	return &Vector{array}
}
