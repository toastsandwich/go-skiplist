// Package skiplist implements a probabilistic skip list, a sorted key-value
// store with expected O(log n) search, insert, and delete performance.
//
// Skip lists maintain multiple levels of linked lists. Lower levels contain
// all elements; higher levels contain fewer elements and serve as shortcuts
// during search. The height of each inserted element is chosen randomly
// using a geometric distribution controlled by the probability parameter P.
//
// Keys ([]byte) are ordered using [bytes.Compare]. Values are also []byte.
//
// The implementation is not concurrent-safe.
//
// Example:
//
//	s := NewSkipList(32, 0.5)
//	s.Push([]byte("alpha"), []byte("first"))
//	s.Push([]byte("beta"), []byte("second"))
//
//	v := s.Get([]byte("alpha")) // []byte("first")
//
//	for k, v := range s.All() {
//	    // iterates in sorted key order
//	}
//
//	s.Pop([]byte("beta"))
package skiplist

import (
	"bytes"
	"errors"
	"iter"
	"math/bits"
	"math/rand/v2"
	"sync"
)

var (
	ErrNilKey      = errors.New("key cannot be nil or zero len slice of byte")
	ErrNilVal      = errors.New("value cannot be nil or zero len slice of byte")
	ErrKeyNotFound = errors.New("key not found")
)

// Return default values for maxlevel and p
func DefaultValues() (int, float64) {
	return 32, 0.5
}

// Element represents one node in the skip list containing a key/value pair.
//
// Level indicates the highest level (0-based) at which this element appears.
// Elements at level L are also linked into all levels 0 through L.
type Element struct {
	// Key and Value hold the stored data. Keys are compared with bytes.Compare.
	Key, Value []byte

	// Level is the highest level this element participates in.
	Level int

	// Unexported forward pointers for each level.
	nexts    []*Element
	nextsLen int
}

// make a header element for skiplist
func makeHeaderElement(max int) *Element {
	e := &Element{
		nexts:    make([]*Element, max),
		nextsLen: max,
	}

	// initially all headers should point to nil
	for i := range e.nexts {
		e.nexts[i] = nilElement
	}
	return e
}

// nilElement is the last element which will just reference
// end of the skiplist, it does not have any next elements
var nilElement = &Element{}

// SetLevel sets the Level of the element. It does not adjust any links.
func (e *Element) SetLevel(l int) {
	e.Level = l
}

// NewElement creates a new Element with the provided key, value and level.
// The created element will have forward pointers allocated up to s.MaxLevel.
//
// This is primarily intended for advanced use cases. Most callers should use
// [SkipList.Push] instead.
func (s *SkipList) NewElement(key, val []byte, l int) *Element {
	return &Element{
		Key:   key,
		Value: val,
		Level: l,
		nexts: make([]*Element, l+1),
	}
}

// SkipList is a sorted collection of []byte key/value pairs implemented as
// a probabilistic skip list.
//
// Header is a sentinel node. Its nexts slice contains the head pointers for
// each level of the skip list.
//
// MaxLevel is the maximum number of levels in the skip list.
//
// P is the probability used when choosing random level for new elements
// (see [NewSkipList]).
type SkipList struct {
	// Header is the entry point for searches at every level.
	Header *Element

	MaxLevel int

	// P controls the probability of promoting an element to a higher level.
	P float64

	nilElement *Element
	len        int
	// used to get update lists
	pool sync.Pool
}

// NewSkipList creates and returns a new SkipList.
//
// maxlevel is the maximum height of the skip list. If <= 0, it defaults to 32.
//
// p is the probability that an inserted element will be promoted to the next
// higher level (geometric distribution). The classic value is 0.5. If p <= 0,
// it defaults to 0.5.
//
// Higher p values produce taller structures on average; lower values produce
// flatter ones. 0.5 is generally a good balance.
func NewSkipList(maxlevel int, p float64) *SkipList {
	if maxlevel <= 0 {
		maxlevel = 32
	}
	if p <= 0 {
		p = 0.5
	}
	return &SkipList{
		Header:     makeHeaderElement(maxlevel),
		MaxLevel:   maxlevel,
		nilElement: nilElement,
		P:          p,
		pool: sync.Pool{
			New: func() any {
				return make([]*Element, maxlevel)
			},
		},
	}
}

// Get returns the value associated with key.
//
// It returns nil if the key does not exist in the skip list.
// Note: because values are []byte, it is not possible to distinguish
// between a missing key and a key whose value is nil or empty using Get alone.
func (s *SkipList) Get(key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, ErrNilKey
	}
	h := s.Header
	for l := s.MaxLevel - 1; l >= 0; l-- {
		for bytes.Compare(h.nexts[l].Key, key) < 0 && h.nexts[l] != s.nilElement {
			h = h.nexts[l]
		}
	}
	h = h.nexts[0]
	if bytes.Equal(h.Key, key) {
		return h.Value, nil
	}
	return nil, ErrKeyNotFound
}

func validateKeyValue(key []byte, value []byte) error {
	if len(key) == 0 {
		return ErrNilKey
	}
	if len(value) == 0 {
		return ErrNilVal
	}
	return nil
}

// Put inserts the key/value pair into the skip list.
// If the key already exists, its value is overwritten.
//
// Elements are maintained in ascending order by key using bytes.Compare.
func (s *SkipList) Put(key, val []byte) error {
	if err := validateKeyValue(key, val); err != nil {
		return err
	}

	update := s.pool.Get().([]*Element)
	defer s.pool.Put(update)

	x := s.Header
	// search for the key
	for i := s.MaxLevel - 1; i >= 0; i-- {
		for bytes.Compare(x.nexts[i].Key, key) < 0 && x.nexts[i] != s.nilElement {
			x = x.nexts[i]
		}
		update[i] = x
	}
	x = x.nexts[0]
	if bytes.Equal(x.Key, key) {
		x.Value = val
		return nil
	}
	lvl := s.randomLevel()
	x = s.NewElement(key, val, lvl)
	for i := 0; i <= lvl; i++ {
		x.nexts[i] = update[i].nexts[i]
		update[i].nexts[i] = x
	}
	s.len++
	return nil
}

// Pop removes the element with the given key and returns its previous value.
// If the key does not exist, Pop returns nil and leaves the list unchanged.
func (s *SkipList) Pop(key []byte) error {
	if len(key) == 0 {
		return ErrNilKey
	}
	update := s.pool.Get().([]*Element)
	defer s.pool.Put(update)

	x := s.Header
	// search for the key
	for i := s.MaxLevel - 1; i >= 0; i-- {
		for bytes.Compare(x.nexts[i].Key, key) < 0 && x.nexts[i] != s.nilElement {
			x = x.nexts[i]
		}
		update[i] = x
	}
	x = x.nexts[0]
	if x == s.nilElement || !bytes.Equal(x.Key, key) {
		return ErrKeyNotFound
	}

	// Unlink the node from all the levels it participates in.
	// A node with Level=l only has forward pointers (and is linked) at levels 0..l.
	for i := 0; i <= x.Level && i < s.MaxLevel; i++ {
		if update[i].nexts[i] == x {
			update[i].nexts[i] = x.nexts[i]
		}
	}

	s.len--
	return nil
}

// All returns an iterator that yields all key/value pairs in ascending
// key order.
//
// It satisfies the iter.Seq2 interface and can be used directly with range:
//
//	for key, value := range list.All() {
//	    ...
//	}
//
// The iterator is valid only for the current state of the list. Modifying
// the list during iteration may produce unexpected results.
func (s *SkipList) All() iter.Seq2[[]byte, []byte] {
	return func(yield func([]byte, []byte) bool) {
		for i := s.Header.nexts[0]; i != s.nilElement; i = i.nexts[0] {
			if !yield(i.Key, i.Value) {
				return
			}
		}
	}
}

func (s *SkipList) ForEach(do func(key, value []byte) bool) {
	for i := s.Header.nexts[0]; i != s.nilElement; i = i.nexts[0] {
		if !do(i.Key, i.Value) {
			return
		}
	}
}

// Returns number of elements in a list
func (s *SkipList) Len() int {
	return s.len
}

func (s *SkipList) randomLevel() int {
	// special case P = 0.5
	// trailing zeros can be considered as level assigned
	// xxxx1 = 0.5   level 0
	// xxx10 = 0.25  level 1
	// xx100 = 0.125 level 2
	if s.P == 0.5 {
		z := bits.TrailingZeros64(rand.Uint64())
		if z >= s.MaxLevel {
			return s.MaxLevel - 1
		}
		return z
	}
	l := 0
	for l < s.MaxLevel-1 && rand.Float64() < s.P {
		l++
	}
	return l
}
