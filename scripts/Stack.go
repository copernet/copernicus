package scripts

type Stack struct {
	stacks            [][]byte
	verifyMinimalData bool
}

func asBool(bytes []byte) bool {
	for i := range bytes {
		byt := bytes[i]
		if byt != 0 {
			if i == len(bytes)-1 && byt == 0x80 {
				return false
			}
			return true
		}
	}
	return false
}

func fromBool(v bool) []byte {
	if v {
		return []byte{1}
	}
	return nil
}

func (stack *Stack) Depth() int32 {
	return int32(len(stack.stacks))
}

func (stack *Stack) PushBytes(bytes []byte) {
	stack.stacks = append(stack.stacks, bytes)
}
