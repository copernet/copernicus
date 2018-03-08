package container

import (
	"sync"
)

// helpful to not write everywhere struct{}{}
var keyExists = struct{}{}

// Set defines a thread safe set data structure.
type Set struct {
	m map[interface{}]struct{} // struct{} doesn't take up space
	l sync.RWMutex             // we name it because we don't want to expose it
}

// NewSet creates and initialize a new Set. It's accept a variable number of
// arguments to populate the initial set. If nothing passed a Set with zero
// size is created.
func NewSet() *Set {
	s := &Set{}
	s.m = make(map[interface{}]struct{})

	return s
}

// AddItems includes the specified items (one or more) to the set. The underlying
// Set s is modified. If passed nothing it silently returns.
func (s *Set) AddItems(items ...interface{}) bool {
	if len(items) == 0 {
		return false
	}

	s.l.Lock()
	defer s.l.Unlock()

	for _, item := range items {
		if _, ok := s.m[item]; ok {
			return false
		}
	}

	for _, item := range items {
		s.m[item] = keyExists
	}

	return true
}

func (s *Set) AddItem(item interface{}) bool {
	s.l.Lock()
	defer s.l.Unlock()

	if _, ok := s.m[item]; ok {
		return false
	}
	s.m[item] = keyExists
	return true
}

// RemoveItems deletes the specified items from the set.  The underlying Set s is
// modified. If passed nothing it silently returns.
func (s *Set) RemoveItems(items ...interface{}) {
	if len(items) == 0 {
		return
	}

	s.l.Lock()
	defer s.l.Unlock()

	for _, item := range items {
		delete(s.m, item)
	}
}

func (s *Set) RemoveItem(item interface{}) bool {
	s.l.Lock()
	defer s.l.Unlock()

	if _, ok := s.m[item]; !ok {
		return ok
	}
	delete(s.m, item)
	return true
}

// Pop  deletes and return an item from the set. The underlying Set s is
// modified. If set is empty, nil is returned.
func (s *Set) Pop() interface{} {
	s.l.RLock()
	for item := range s.m {
		s.l.RUnlock()
		s.l.Lock()
		delete(s.m, item)
		s.l.Unlock()
		return item
	}
	s.l.RUnlock()
	return nil
}

// HasItems looks for the existence of items passed. It returns false if nothing is
// passed. For multiple items it returns true only if all of  the items exist.
func (s *Set) HasItems(items ...interface{}) bool {
	// assume checked for empty item, which not exist
	if len(items) == 0 {
		return false
	}

	s.l.RLock()
	defer s.l.RUnlock()

	has := true
	for _, item := range items {
		if _, has = s.m[item]; !has {
			break
		}
	}
	return has
}

// Size returns the number of items in a set.
func (s *Set) Size() int {
	s.l.RLock()
	defer s.l.RUnlock()

	l := len(s.m)
	return l
}

// Clear removes all items from the set.
func (s *Set) Clear() {
	s.l.Lock()
	defer s.l.Unlock()

	s.m = make(map[interface{}]struct{})
}

// IsEqual test whether s and t are the same in size and have the same items.
func (s *Set) IsEqual(t *Set) bool {
	s.l.RLock()
	t.l.RLock()
	defer s.l.RUnlock()
	defer t.l.RUnlock()

	// return false if they are no the same size
	if sameSize := len(s.m) == t.Size(); !sameSize {
		return false
	}

	equal := true
	t.Each(func(item interface{}) bool {
		_, equal = s.m[item]
		return equal // if false, Each() will end
	})

	return equal
}

// IsSubset tests whether t is a subset of s.
func (s *Set) IsSubset(t *Set) (subset bool) {
	s.l.RLock()
	defer s.l.RUnlock()

	subset = true

	t.Each(func(item interface{}) bool {
		_, subset = s.m[item]
		return subset
	})

	return
}

// Each traverses the items in the Set, calling the provided function for each
// set member. Traversal will continue until all items in the Set have been
// visited, or if the closure returns false.
func (s *Set) Each(f func(item interface{}) bool) {
	s.l.RLock()
	defer s.l.RUnlock()

	for item := range s.m {
		if !f(item) {
			break
		}
	}
}

// Copy returns a new Set with a copy of s.
func (s *Set) Copy() *Set {
	ret := &Set{}
	ret.m = make(map[interface{}]struct{})
	s.Each(func(item interface{}) bool {
		ret.AddItem(item)
		return true
	})
	return ret
}

// Merge is like Union, however it modifies the current set it's applied on
// with the given t set.
func (s *Set) Merge(t *Set) {
	s.l.Lock()
	defer s.l.Unlock()

	t.Each(func(item interface{}) bool {
		s.m[item] = keyExists
		return true
	})
}

// IsEmpty reports whether the Set is empty.
func (s *Set) IsEmpty() bool {
	return s.Size() == 0
}

// List returns a slice of all items.
func (s *Set) List() []interface{} {
	s.l.RLock()
	defer s.l.RUnlock()

	list := make([]interface{}, 0, len(s.m))

	for item := range s.m {
		list = append(list, item)
	}

	return list
}
