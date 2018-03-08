package container

import (
	"fmt"
	"sort"
)

type CmpFunc func(a interface{}, b interface{}) bool

type CacheMap struct {
	m       map[interface{}]interface{}
	keys    []interface{}
	camFunc CmpFunc
}

func (cacheMap *CacheMap) Len() int {
	return len(cacheMap.m)
}

func (cacheMap *CacheMap) Less(i, j int) bool {
	return cacheMap.camFunc(cacheMap.keys[i], cacheMap.keys[j])
}

func (cacheMap *CacheMap) Swap(i, j int) {
	cacheMap.keys[i], cacheMap.keys[j] = cacheMap.keys[j], cacheMap.keys[i]

}

func (cacheMap *CacheMap) Add(key interface{}, value interface{}) {
	if _, ok := cacheMap.m[key]; !ok {
		cacheMap.keys = append(cacheMap.keys, key)
		cacheMap.m[key] = value
		sort.Sort(cacheMap)
	} else {
		cacheMap.m[key] = value
	}
}

func (cacheMap *CacheMap) AddElementToSlice(key interface{}, value interface{}) {
	if v, ok := cacheMap.m[key]; !ok {
		valueSlice := make([]interface{}, 0)
		valueSlice = append(valueSlice, value)
		cacheMap.m[key] = valueSlice
		cacheMap.keys = append(cacheMap.keys, key)
		sort.SliceStable(cacheMap.keys, func(i, j int) bool {
			return cacheMap.camFunc(cacheMap.keys[i], cacheMap.keys[j])
		})
	} else {
		valueSlice := v.([]interface{})
		valueSlice = append(valueSlice, value)
		cacheMap.m[key] = valueSlice
	}
}

func (cacheMap *CacheMap) Del(key interface{}) {
	keys := make([]interface{}, 0)
	for _, cacheKey := range cacheMap.keys {
		if cacheKey != key {
			keys = append(keys, cacheKey)
		}
	}
	cacheMap.keys = keys
	m := make(map[interface{}]interface{})
	for k, v := range cacheMap.m {
		if k != key {
			m[k] = v
		}
	}
	cacheMap.m = m
	sort.SliceStable(cacheMap.keys, func(i, j int) bool {
		return cacheMap.camFunc(cacheMap.keys[i], cacheMap.keys[j])
	})

}

func (cacheMap *CacheMap) GetLowerBoundKey(key interface{}) interface{} {
	i := sort.Search(len(cacheMap.keys), func(i int) bool {
		return cacheMap.camFunc(cacheMap.keys[1], key)
	})
	if i >= len(cacheMap.keys) {
		return nil
	}
	return cacheMap.keys[i]
}

func (cacheMap *CacheMap) GetMap() map[interface{}]interface{} {
	return cacheMap.m

}
func (cacheMap *CacheMap) Get(key interface{}) interface{} {
	return cacheMap.m[key]
}

func (cacheMap *CacheMap) First() interface{} {
	if cacheMap.Len() == 0 {
		return nil
	}
	return cacheMap.m[cacheMap.keys[0]]
}

func (cacheMap *CacheMap) Last() interface{} {
	if cacheMap.Len() == 0 {
		return nil
	}
	return cacheMap.m[cacheMap.keys[cacheMap.Len()-1]]
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

func (cacheMap *CacheMap) GetAllKeys() []interface{} {
	return cacheMap.keys
}

func (cacheMap *CacheMap) Size() int {
	return len(cacheMap.keys)
}

func NewCacheMap(camFunc CmpFunc) *CacheMap {
	m := make(map[interface{}]interface{})
	keys := make([]interface{}, 0)
	cacheMap := CacheMap{m: m, keys: keys, camFunc: camFunc}
	return &cacheMap
}
