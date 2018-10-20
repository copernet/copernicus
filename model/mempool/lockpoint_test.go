package mempool

import (
	"testing"
	"reflect"
)

func TestNewLockPoints(t *testing.T) {
	lockpoint := &LockPoints{
		Height:        0,
		Time:          0,
		MaxInputBlock: nil,
	}

	newLockPoint := NewLockPoints()
	if !reflect.DeepEqual(lockpoint, newLockPoint) {
		t.Errorf("NewLockPoints failed, newLockPoint is:%v", newLockPoint)
	}
}
