package algorithm

import (
	"container/list"
	"github.com/pkg/errors"
	"sync"
)

type KeyValue struct {
	Key, Value interface{}
}
type LRUCache struct {
	lock     sync.Mutex
	Length   int
	dataList *list.List
	cacheMap map[interface{}]*list.Element
}

func NewLRUCache(length int) (*LRUCache) {

	return &LRUCache{Length: length,
		dataList:        list.New(),
		cacheMap:        make(map[interface{}]*list.Element),
	}
}
func (lru *LRUCache) Size() (int) {
	lru.lock.Lock()
	defer lru.lock.Unlock()
	return lru.dataList.Len()

}

func (lru *LRUCache) Add(k, v interface{}) (error) {
	lru.lock.Lock()
	defer lru.lock.Unlock()

	if lru.dataList == nil {
		return errors.New("lurCache is nil")
	}
	if element, exist := lru.cacheMap[k]; exist {
		lru.dataList.MoveToFront(element)
		element.Value = v
		return nil
	}
	node := lru.dataList.PushFront(v)
	lru.cacheMap[k] = node
	if lru.dataList.Len() > lru.Length {
		lastElement := lru.dataList.Back()
		if lastElement == nil {
			return nil
		}
		delete(lru.cacheMap, k)
		lru.dataList.Remove(lastElement)

	}
	return nil

}

func (lru *LRUCache) Get(k interface{}) (interface{}, bool, error) {
	lru.lock.Lock()
	defer lru.lock.Unlock()
	if lru.cacheMap == nil {
		return nil, false, errors.New("LruCache is nil ")
	}
	if element, exist := lru.cacheMap[k]; exist {
		lru.dataList.MoveToFront(element)
		return element.Value, true, nil
	}
	return nil, false, nil
}

func (lru *LRUCache) Remove(k interface{}) (bool) {
	lru.lock.Lock()
	defer lru.lock.Unlock()
	if lru.cacheMap == nil {
		return false
	}
	if element, exist := lru.cacheMap[k]; exist {
		lru.dataList.Remove(element)
		delete(lru.cacheMap, k)
		return true

	}
	return false
}
