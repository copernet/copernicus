package algorithm

import (
	"testing"
)

func com(a, b interface{}) bool {
	return a.(int) > b.(int)
}

func TestCustomSetAddInterm(t *testing.T) {
	s := NewCustomSet(com)
	for i := 0; i < 10; i++ {
		if ok := s.AddInterm(i); !ok {
			t.Error("add item to set error")
			return
		}
	}

	j := 0
	for i := len(s.sortData); i > 0; i-- {
		if s.sortData[j].(int) != i-1 {
			t.Errorf("the two element should equal..., origin : %d, expect : %d\n", s.sortData[j], i-1)
			return
		}
		j++
	}

	if ok := s.DelItem(3); !ok {
		t.Errorf("delete item : %d\n", 3)
		return
	}
	if s.sortData[5].(int) != 4 {
		t.Errorf("index : %d, origin : %d, expect : %d\n", 6, s.sortData[5].(int), 4)
		return
	}

	if !s.HasItem(2) {
		t.Errorf("should have item : 2")
		return
	}

	if !s.DelItemByIndex(6) {
		t.Errorf("should delete index 6 item")
		return
	}
	if s.sortData[6] != 1 {
		t.Errorf("index : %d, origin : %d, expect : %d\n", 6, s.sortData[6].(int), 1)
		return
	}
	if s.DelItemByIndex(-10) {
		t.Errorf("no index equal -10")
		return
	}
	if s.DelItemByIndex(100) {
		t.Errorf("no index equal 100")
		return
	}
	if s.DelItem(198) {
		t.Errorf("set no have item equal 198 ")
		return
	}
	if s.Size() != 8 {
		t.Errorf("the set should have size : 8, but actual have : %d\n", s.Size())
		return
	}
	if s.Begin() != 9 {
		t.Errorf("the set begin item : %d, but actual value : %d\n", 9, 9)
		return
	}
	if s.End() != 0 {
		t.Errorf("the set end item : %d, but actual value : %d\n", 0, 0)
		return
	}

}
