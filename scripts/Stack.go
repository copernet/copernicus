package scripts

import (
	"encoding/hex"
	"github.com/pkg/errors"
)

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

func (stack *Stack) PushInt(scriptNum ScriptNum) {
	stack.PushBytes(scriptNum.Bytes())
}

func (stack *Stack) PushBool(val bool) {
	stack.PushBytes(fromBool(val))
}

func (stack *Stack) PopByteArray() ([]byte, error) {
	return stack.nipN(0)
}

// PopInt pops the value off the top of the stack, converts it into a script
// num, and returns it.  The act of converting to a script num enforces the
// consensus rules imposed on data interpreted as numbers.
//
// Stack transformation: [... x1 x2 x3] -> [... x1 x2]
func (stack *Stack) PopInt() (ScriptNum, error) {
	so, err := stack.PopByteArray()
	if err != nil {
		return 0, err
	}
	return makeScriptNum(so, stack.verifyMinimalData, DefaultScriptNumLen)
}

// PopBool pops the value off the top of the stack, converts it into a bool, and
// returns it.
//
// Stack transformation: [... x1 x2 x3] -> [... x1 x2]
func (stack *Stack) PopBool() (bool, error) {
	so, err := stack.PopByteArray()
	if err != nil {
		return false, err
	}
	return asBool(so), nil

}

// PeekByteArray returns the Nth item on the stack without removing it.
func (stack *Stack) PeekByteArray(idx int32) ([]byte, error) {
	stackLen := int32(len(stack.stacks))
	if idx < 0 || idx >= stackLen {
		return nil, errors.Errorf("index %d is invalid for stack size %d", idx, stackLen)
	}
	return stack.stacks[stackLen-idx-1], nil
}

func (stack *Stack) PeekInt(idx int32) (ScriptNum, error) {
	so, err := stack.PeekByteArray(idx)
	if err != nil {
		return 0, err
	}
	return makeScriptNum(so, stack.verifyMinimalData, DefaultScriptNumLen)
}
func (stack *Stack) PeekBool(idx int32) (bool, error) {
	so, err := stack.PeekByteArray(idx)
	if err != nil {
		return false, err
	}
	return asBool(so), nil
}

// nipN is an internal function that removes the nth item on the stack and
// returns it.
//
// Stack transformation:
// nipN(0): [... x1 x2 x3] -> [... x1 x2]
// nipN(1): [... x1 x2 x3] -> [... x1 x3]
// nipN(2): [... x1 x2 x3] -> [... x2 x3]
func (stack *Stack) nipN(idx int32) ([]byte, error) {
	stackLen := int32(len(stack.stacks))
	if idx < 0 || idx > stackLen-1 {
		return nil, errors.Errorf("index %d is invalid for stack size %d", idx, stackLen)
	}
	so := stack.stacks[stackLen-idx-1]
	if idx == 0 {
		stack.stacks = stack.stacks[:stackLen-1]
	} else if idx == stackLen-1 {
		s1 := make([][]byte, stackLen-1)
		copy(s1, stack.stacks[1:])
		stack.stacks = s1

	} else {
		s1 := stack.stacks[stackLen-idx : stackLen]
		stack.stacks = stack.stacks[:stackLen-idx-1]
		stack.stacks = append(stack.stacks, s1...)
	}
	return so, nil
}

func (stack *Stack) NipN(idx int32) error {
	_, err := stack.nipN(idx)
	return err
}

// Tuck copies the item at the top of the stack and inserts it before the 2nd
// to top item.
//
// Stack transformation: [... x1 x2] -> [... x2 x1 x2]
func (stack *Stack) Tuck() error {
	so2, err := stack.PopByteArray()
	if err != nil {
		return err
	}
	so1, err := stack.PopByteArray()
	if err != nil {
		return err
	}
	stack.PushBytes(so2)
	stack.PushBytes(so1)
	stack.PushBytes(so2)
	return nil

}

// DropN removes the top N items from the stack.
//
// Stack transformation:
// DropN(1): [... x1 x2] -> [... x1]
// DropN(2): [... x1 x2] -> [...]
func (stack *Stack) DropN(n int32) error {
	if n < 1 {
		return errors.Errorf("attempt to drop %d items from stack", n)
	}
	for ; n > 0; n-- {
		_, err := stack.PopByteArray()
		if err != nil {
			return err
		}
	}
	return nil
}

// DuplicateN duplicates the top N items on the stack.
//
// Stack transformation:
// DupN(1): [... x1 x2] -> [... x1 x2 x2]
// DupN(2): [... x1 x2] -> [... x1 x2 x1 x2]
func (stack *Stack) DuplicateN(n int32) error {
	if n < 1 {
		return errors.Errorf("attempt to duplicate %d stack items", n)
	}
	// Iteratively duplicate the value n-1 down the stack n times.
	// This leaves an in-order duplicate of the top n items on the stack.
	for i := n; i > 0; i-- {
		so, err := stack.PeekByteArray(n - 1)
		if err != nil {
			return err
		}
		stack.PushBytes(so)
	}
	return nil

}

// RotN rotates the top 3N items on the stack to the left N times.
//
// Stack transformation:
// RotN(1): [... x1 x2 x3] -> [... x2 x3 x1]
// RotN(2): [... x1 x2 x3 x4 x5 x6] -> [... x3 x4 x5 x6 x1 x2]

func (stack *Stack) RotateN(n int32) error {
	if n < 1 {
		return errors.Errorf("attempt to rotate %d stack items", n)
	}
	entry := 3*n - 1
	for i := n; i > 0; i-- {
		so, err := stack.nipN(entry)
		if err != nil {
			return err
		}
		stack.PushBytes(so)
	}
	return nil

}

// SwapN swaps the top N items on the stack with those below them.
//
// Stack transformation:
// SwapN(1): [... x1 x2] -> [... x2 x1]
// SwapN(2): [... x1 x2 x3 x4] -> [... x3 x4 x1 x2]
func (stack *Stack) SwapN(n int32) error {
	if n < 1 {
		return errors.Errorf("attempt to swap %d stack items", n)
	}
	entry := 2*n - 1
	for i := n; i > 0; i-- {
		so, err := stack.nipN(entry)
		if err != nil {
			return err
		}
		stack.PushBytes(so)
	}
	return nil
}

// OverN copies N items N items back to the top of the stack.
//
// Stack transformation:
// OverN(1): [... x1 x2 x3] -> [... x1 x2 x3 x2]
// OverN(2): [... x1 x2 x3 x4] -> [... x1 x2 x3 x4 x1 x2]
func (stack *Stack) OverN(n int32) error {
	if n < 1 {
		return errors.Errorf("attempt to perform over on %d stack items", n)
	}
	entry := 2*n - 1
	for ; n > 0; n-- {
		so, err := stack.PeekByteArray(entry)
		if err != nil {
			return err
		}
		stack.PushBytes(so)
	}
	return nil

}

// PickN copies the item N items back in the stack to the top.
//
// Stack transformation:
// PickN(0): [x1 x2 x3] -> [x1 x2 x3 x3]
// PickN(1): [x1 x2 x3] -> [x1 x2 x3 x2]
// PickN(2): [x1 x2 x3] -> [x1 x2 x3 x1]
func (stack *Stack) PickN(n int32) error {
	so, err := stack.PeekByteArray(n)
	if err != nil {
		return err
	}
	stack.PushBytes(so)
	return nil
}

// RollN moves the item N items back in the stack to the top.
//
// Stack transformation:
// RollN(0): [x1 x2 x3] -> [x1 x2 x3]
// RollN(1): [x1 x2 x3] -> [x1 x3 x2]
// RollN(2): [x1 x2 x3] -> [x2 x3 x1]
func (stack *Stack) RollN(n int32) error {
	so, err := stack.nipN(n)
	if err != nil {
		return err
	}
	stack.PushBytes(so)
	return nil
}
func (stack *Stack) String() string {
	var result string
	for _, stack := range stack.stacks {
		if len(stack) == 0 {
			result += "00000000  <empty>\n"
		}
		result += hex.Dump(stack)
	}
	return result

}
