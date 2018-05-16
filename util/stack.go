package util

type Stack struct {
	array []interface{}
}

func (s *Stack) Size() int {
	return len(s.array)
}

func (s *Stack) Empty() bool {
	return len(s.array) == 0
}

func (s *Stack) List() []interface{} {
	return s.array
}

func (s *Stack) Copy() *Stack {
	bak := make([]interface{}, s.Size())
	copy(bak, s.array)

	return &Stack{
		array: bak,
	}
}

func (s *Stack) PushStack(value interface{}) {
	s.array = append(s.array, value)
}

func (s *Stack) Swap(i int, j int) bool {
	if i > s.Size() - 1 || i < 0 {
		return false
	}
	if j > s.Size() - 1 || j < 0 {
		return false
	}
	s.array[i], s.array[j] = s.array[j], s.array[i]

	return true

}
func (s *Stack) PopStack() (interface{}) {
	stackLen := len(s.array)
	if stackLen == 0 {
		return nil
	}
	e := s.array[stackLen - 1]
	if e == nil {
		return nil
	}
	s.array = s.array[:stackLen - 1]

	return e
}
func (s *Stack) RemoveAt(index int) bool {
	if index > s.Size() - 1 || index < 0 {
		return false
	}
	s.array = append(s.array[:index], s.array[index + 1:])

	return true
}

func (s *Stack) Last() interface{} {
	if s.Size() == 0 {
		return nil
	}
	return s.array[s.Size() - 1]
}

func (s *Stack) Erase(begin int, end int) bool {
	size := s.Size()
	if begin < 0 || end < 0 || begin >= end || begin > size || end > size {
		return false
	}
	for i := begin; i < end; i++ {
		if !s.RemoveAt(i) {
			return false
		}
	}
	return true
}

func (s *Stack) Insert(index int, value interface{}) bool {
	if index > s.Size() - 1 || index < 0 {
		return false
	}
	lastArray := s.array[index + 1:]
	s.array = append(s.array[:index], value)
	if len(lastArray) > 0 {
		s.array = append(s.array, lastArray...)
	}
	return true
}

func (s *Stack) StackTop(i int) (interface{}) {
	stackLen := s.Size()
	if stackLen + i > stackLen - 1 || stackLen + i < 0{
		return nil
	}
	return s.array[stackLen + i]
}

func (s *Stack) Equal(other *Stack) bool {
	if s.Size() != other.Size() {
		return false
	}
	for i := 0; i < s.Size(); i++ {
		if s.array[i] != other.array[i] {
			return false
		}
	}
	return true
}


func SwapStack(s *Stack, other *Stack) {
	if s.Size() == 0 && other.Size() == 0 {
		return
	}
	if other.Size() == 0 {
		other.array = s.array[:s.Size()]
		s.array = make([]interface{}, 0)
	}
	if s.Size() == 0 {
		s.array = other.array[:other.Size()]
		other.array = make([]interface{}, 0)
	}
	s.array, other.array = other.array, s.array

}

func CopyStackByteType(des *Stack, src *Stack) {
	if src.Size() == 0 {
		return
	}
	for _, v := range src.array {
		switch element := v.(type) {
		case []byte:
			{
				length := len(element)
				tmpSlice := make([]byte, length)
				copy(tmpSlice, element)
				des.array = append(des.array, tmpSlice)
			}
		default:

		}
	}
}

func NewStack() *Stack {
	array := make([]interface{}, 0)
	return &Stack{array}
}
