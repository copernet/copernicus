package peer

import (
	"sync"
	"container/list"
	"bytes"
	"fmt"
)

type NonceMap struct {
	lock     sync.Mutex
	nonceMap map[uint64]*list.Element
	limit    uint
}

func (m *NonceMap) ToString() string {
	m.lock.Lock()
	defer m.lock.Unlock()
	length := len(m.nonceMap)
	index := 0
	buf := bytes.NewBufferString("[")
	for nonce := range m.nonceMap {
		buf.WriteString(fmt.Sprintf("%d", nonce))
		if index < length-1 {
			buf.WriteString(",")
		}
		index++
	}
	buf.WriteString("]")
	return fmt.Sprintf("<%d>%s", length, buf.String())
}

func (m*NonceMap) Exists(nonce uint64) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	if _, exists := m.nonceMap[nonce]; exists {
		return true
	}
	return false
}

func (m*NonceMap) Add(nonce uint64) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.limit==0{
		return
	}
	if node,exists :=m.nonceMap[nonce];exists{
		nonce= uint64(node)
		return
	}
	if uint(len(m.nonceMap))+1>m.limit{
		lru:=m.nonceMap[]

	}

}
