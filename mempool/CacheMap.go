package mempool

import (
	"fmt"
	"sort"
)

type ISortKey interface {
	Cmp(other ISortKey) int
}
type CacheMap struct {
	m    map[ISortKey]interface{}
	keys []ISortKey
}

func (cacheMap *CacheMap) Len() int {
	return len(cacheMap.m)
}

func (cacheMap *CacheMap) Less(i, j int) bool {
	return cacheMap.keys[i].Cmp(cacheMap.keys[j]) > 0
}

func (cacheMap *CacheMap) Swap(i, j int) {
	cacheMap.keys[i], cacheMap.keys[j] = cacheMap.keys[j], cacheMap.keys[i]

}

func (cacheMap *CacheMap) Add(key ISortKey, value interface{}) {
	cacheMap.keys = append(cacheMap.keys, key)
	cacheMap.m[key] = value
	sort.Sort(cacheMap)
}

func (cacheMap *CacheMap) Del(key ISortKey) {
	keys := make([]ISortKey, 0)
	for _, cacheKey := range cacheMap.keys {
		if cacheKey != key {
			keys = append(keys, cacheKey)
		}
	}
	cacheMap.keys = keys
	m := make(map[ISortKey]interface{})
	for k, v := range cacheMap.m {
		if k != key {
			m[k] = v
		}
	}
	cacheMap.m = m
	sort.Sort(cacheMap)

}

func (cacheMap *CacheMap) Get(key ISortKey) interface{} {
	return cacheMap.m[key]
}

func (cacheMap *CacheMap) String() string {
	len := cacheMap.Len()
	if len == 0 {
		return ""
	}
	queryStr := ""
	for k, v := range cacheMap.m {
		queryStr = fmt.Sprintf("%s/n %s=%s", queryStr, k, v)

	}
	return queryStr

}

func (cacheMap *CacheMap) GetAllKeys() []ISortKey {
	return cacheMap.keys
}
