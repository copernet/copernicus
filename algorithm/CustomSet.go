package algorithm

import "sort"

type CompareFunction func(a, b interface{}) bool

type CustomSet struct {
	data     map[interface{}]struct{}
	sortData []interface{}
	comFun   CompareFunction
}

func NewCustomSet(c CompareFunction) *CustomSet {
	s := CustomSet{}
	s.data = make(map[interface{}]struct{})
	s.comFun = c

	return &s
}

func (c *CustomSet) AddInterm(item interface{}) bool {
	if _, ok := c.data[item]; ok {
		return false
	}
	c.data[item] = struct{}{}
	c.sortData = append(c.sortData, item)
	sort.SliceStable(c.sortData, func(i, j int) bool {
		return c.comFun(c.sortData[i], c.sortData[j])
	})

	return true
}

func (c *CustomSet) DelItem(item interface{}) bool {
	if _, ok := c.data[item]; !ok {
		return false
	}
	delete(c.data, item)
	tem := make([]interface{}, len(c.data))
	i := 0
	for _, v := range c.sortData {
		if v != item {
			tem[i] = v
			i++
		}
	}
	c.sortData = tem
	return true
}

func (c *CustomSet) DelItemByIndex(index int) bool {
	if index < 0 || index >= len(c.sortData) {
		return false
	}
	k := c.sortData[index]
	delete(c.data, k)
	temp := make([]interface{}, len(c.data))
	copy(temp, c.sortData[:index])
	copy(temp[index:], c.sortData[index+1:])
	c.sortData = temp
	return true
}

func (c *CustomSet) GetItemByIndex(index int) interface{} {
	if index < 0 || index >= len(c.sortData) {
		return nil
	}
	return c.sortData[index]
}

func (c *CustomSet) Size() int {
	return len(c.sortData)
}

// GetItemIndex when item not in set, return -1;
func (c *CustomSet) GetItemIndex(item interface{}) int {
	if _, ok := c.data[item]; !ok {
		return -1
	}
	n := 0
	sort.Search(len(c.sortData), func(i int) bool {
		ret := c.comFun(c.sortData[i], item)
		if ret {
			n = i
		}
		return ret
	})
	return n
}

func (c *CustomSet) Begin() interface{} {
	if len(c.sortData) > 0 {
		return c.sortData[0]
	}
	return nil
}

func (c *CustomSet) End() interface{} {
	if len(c.sortData) > 0 {
		return c.sortData[len(c.sortData)-1]
	}
	return nil
}

func (c *CustomSet) HasItem(item interface{}) bool {
	if _, ok := c.data[item]; ok {
		return ok
	}
	return false
}
