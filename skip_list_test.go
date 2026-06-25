package skiplist

import (
	"bytes"
	"fmt"
	"testing"
)

func TestNewSkipListDefaults(t *testing.T) {
	s := NewSkipList(0, 0)
	if s.MaxLevel != 32 {
		t.Errorf("expected MaxLevel=32, got %d", s.MaxLevel)
	}
	if s.P != 0.5 {
		t.Errorf("expected P=0.5, got %f", s.P)
	}

	s2 := NewSkipList(-5, -1)
	if s2.MaxLevel != 32 {
		t.Errorf("expected default MaxLevel=32 for negative, got %d", s2.MaxLevel)
	}
	if s2.P != 0.5 {
		t.Errorf("expected default P=0.5 for negative, got %f", s2.P)
	}
}

func TestPushGet(t *testing.T) {
	s := NewSkipList(16, 0.5)

	s.Push([]byte("hello"), []byte("world"))
	val := s.Get([]byte("hello"))
	if !bytes.Equal(val, []byte("world")) {
		t.Errorf("expected world, got %q", val)
	}

	// empty value
	s.Push([]byte("emptyval"), []byte{})
	if !bytes.Equal(s.Get([]byte("emptyval")), []byte{}) {
		t.Error("expected empty value")
	}

	// empty key
	s.Push([]byte{}, []byte("emptykey"))
	if !bytes.Equal(s.Get([]byte{}), []byte("emptykey")) {
		t.Error("failed empty key")
	}
}

func TestPushUpdate(t *testing.T) {
	s := NewSkipList(16, 0.5)
	s.Push([]byte("k1"), []byte("v1"))
	s.Push([]byte("k1"), []byte("v2"))

	if !bytes.Equal(s.Get([]byte("k1")), []byte("v2")) {
		t.Error("update did not replace value")
	}
}

func TestGetMissing(t *testing.T) {
	s := NewSkipList(16, 0.5)
	if s.Get([]byte("nope")) != nil {
		t.Error("expected nil for missing key")
	}
	if s.Get(nil) != nil {
		t.Error("expected nil for nil key on empty list")
	}
}

func TestPop(t *testing.T) {
	s := NewSkipList(16, 0.5)
	s.Push([]byte("a"), []byte("1"))
	s.Push([]byte("b"), []byte("2"))
	s.Push([]byte("c"), []byte("3"))

	v := s.Pop([]byte("b"))
	if !bytes.Equal(v, []byte("2")) {
		t.Errorf("expected 2, got %q", v)
	}

	if s.Get([]byte("b")) != nil {
		t.Error("key b should be gone after pop")
	}

	// remaining keys still there
	if !bytes.Equal(s.Get([]byte("a")), []byte("1")) {
		t.Error("a missing")
	}
	if !bytes.Equal(s.Get([]byte("c")), []byte("3")) {
		t.Error("c missing")
	}
}

func TestPopMissing(t *testing.T) {
	s := NewSkipList(16, 0.5)
	s.Push([]byte("x"), []byte("y"))

	if s.Pop([]byte("z")) != nil {
		t.Error("pop missing should return nil")
	}
	if s.Pop(nil) != nil {
		t.Error("pop nil should return nil")
	}
}

func TestPopAll(t *testing.T) {
	s := NewSkipList(8, 0.5)
	keys := [][]byte{[]byte("1"), []byte("2"), []byte("3")}
	for _, k := range keys {
		s.Push(k, append([]byte{}, k...))
	}
	for _, k := range keys {
		s.Pop(k)
	}
	for _, k := range keys {
		if s.Get(k) != nil {
			t.Errorf("expected all popped, still have %q", k)
		}
	}
	count := 0
	for range s.All() {
		count++
	}
	if count != 0 {
		t.Errorf("expected 0 elements, got %d", count)
	}
}

func TestAll(t *testing.T) {
	s := NewSkipList(16, 0.5)

	// insert out of order
	s.Push([]byte("c"), []byte("3"))
	s.Push([]byte("a"), []byte("1"))
	s.Push([]byte("b"), []byte("2"))
	s.Push([]byte("d"), []byte("4"))

	var gotKeys [][]byte
	var gotVals [][]byte
	for k, v := range s.All() {
		gotKeys = append(gotKeys, append([]byte{}, k...))
		gotVals = append(gotVals, append([]byte{}, v...))
	}

	wantKeys := [][]byte{[]byte("a"), []byte("b"), []byte("c"), []byte("d")}
	for i := range wantKeys {
		if !bytes.Equal(gotKeys[i], wantKeys[i]) {
			t.Errorf("key order wrong at %d: got %q want %q", i, gotKeys[i], wantKeys[i])
		}
	}
	if !bytes.Equal(gotVals[1], []byte("2")) {
		t.Error("value mismatch in iterator")
	}
}

func TestAllEmpty(t *testing.T) {
	s := NewSkipList(16, 0.5)
	count := 0
	for range s.All() {
		count++
	}
	if count != 0 {
		t.Errorf("expected 0 on empty, got %d", count)
	}
}

func TestCorrectnessLarge(t *testing.T) {
	const N = 2000
	s := NewSkipList(32, 0.5)

	// Use fixed-width keys so lexical order matches numeric
	keys := make([][]byte, N)
	vals := make([][]byte, N)
	for i := 0; i < N; i++ {
		k := []byte(fmt.Sprintf("key-%08d", i))
		keys[i] = k
		vals[i] = []byte(fmt.Sprintf("val-%08d", i))
		s.Push(keys[i], vals[i])
	}

	// All gets
	for i := 0; i < N; i++ {
		got := s.Get(keys[i])
		if !bytes.Equal(got, vals[i]) {
			t.Fatalf("get mismatch at %d", i)
		}
	}

	// Verify sorted iteration and full count
	i := 0
	var prev []byte
	for k, v := range s.All() {
		if prev != nil && bytes.Compare(prev, k) >= 0 {
			t.Fatalf("not sorted at index %d: %q >= %q", i, prev, k)
		}
		if !bytes.Equal(v, vals[i]) {
			t.Fatalf("iter value mismatch at %d", i)
		}
		prev = append([]byte(nil), k...)
		i++
	}
	if i != N {
		t.Fatalf("expected %d elements in iter, got %d", N, i)
	}
}

func TestMixedOperations(t *testing.T) {
	s := NewSkipList(16, 0.5)

	ops := []struct {
		op  string
		k   string
		v   string
		exp string // expected get after
	}{
		{"push", "k1", "v1", "v1"},
		{"push", "k2", "v2", "v2"},
		{"push", "k1", "v1-up", "v1-up"},
		{"pop", "k2", "", ""},
		{"get", "k2", "", ""}, // missing
		{"push", "k3", "v3", "v3"},
		{"pop", "k1", "", ""},
		{"get", "k1", "", ""},
	}

	for _, o := range ops {
		switch o.op {
		case "push":
			s.Push([]byte(o.k), []byte(o.v))
		case "pop":
			s.Pop([]byte(o.k))
		}
		got := s.Get([]byte(o.k))
		if o.exp == "" {
			if got != nil {
				t.Errorf("after %s %s expected missing, got %q", o.op, o.k, got)
			}
		} else if !bytes.Equal(got, []byte(o.exp)) {
			t.Errorf("after %s %s expected %s got %q", o.op, o.k, o.exp, got)
		}
	}
}

func TestUpdateDoesNotAffectOrder(t *testing.T) {
	s := NewSkipList(16, 0.5)
	s.Push([]byte("b"), []byte("2"))
	s.Push([]byte("a"), []byte("1"))
	s.Push([]byte("c"), []byte("3"))
	s.Push([]byte("a"), []byte("1-updated"))

	var keys []string
	for k := range s.All() {
		keys = append(keys, string(k))
	}
	if len(keys) != 3 || keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Errorf("order broken after update: %v", keys)
	}
}

// Benchmarks

func BenchmarkSkipList_Push(b *testing.B) {
	b.ReportAllocs()
	keys := make([][]byte, b.N)
	for i := range keys {
		keys[i] = []byte(fmt.Sprintf("key-%08d", i))
	}
	s := NewSkipList(32, 0.5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Push(keys[i], keys[i])
	}
}

func BenchmarkSkipList_Get(b *testing.B) {
	b.ReportAllocs()
	const n = 10000
	s := NewSkipList(32, 0.5)
	keys := make([][]byte, n)
	for i := 0; i < n; i++ {
		keys[i] = []byte(fmt.Sprintf("key-%08d", i))
		s.Push(keys[i], keys[i])
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Get(keys[i%n])
	}
}

func BenchmarkSkipList_Pop(b *testing.B) {
	b.ReportAllocs()
	// Measure pop cost in steady state: pop + reinsert to keep dataset size constant
	const n = 5000
	keys := make([][]byte, n)
	for i := range keys {
		keys[i] = []byte(fmt.Sprintf("key-%08d", i))
	}
	s := NewSkipList(32, 0.5)
	for j := range keys {
		s.Push(keys[j], keys[j])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		k := keys[i%n]
		s.Pop(k)
		s.Push(k, k) // restore for next iteration
	}
}

func BenchmarkSkipList_Iteration(b *testing.B) {
	b.ReportAllocs()
	const n = 10000
	s := NewSkipList(32, 0.5)
	for i := 0; i < n; i++ {
		k := []byte(fmt.Sprintf("key-%08d", i))
		s.Push(k, k)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count := 0
		for range s.All() {
			count++
		}
		_ = count
	}
}

func BenchmarkSkipList_PushGetMix(b *testing.B) {
	b.ReportAllocs()
	s := NewSkipList(32, 0.5)
	// Preload some data
	for i := 0; i < 1000; i++ {
		k := []byte(fmt.Sprintf("preload-%08d", i))
		s.Push(k, k)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		k := []byte(fmt.Sprintf("key-%08d", i))
		s.Push(k, k)
		_ = s.Get(k)
	}
}

// Example demonstrates basic usage of SkipList.
func Example() {
	s := NewSkipList(16, 0.5)

	s.Push([]byte("cat"), []byte("meow"))
	s.Push([]byte("dog"), []byte("woof"))
	s.Push([]byte("cat"), []byte("purr")) // update

	fmt.Println("cat ->", string(s.Get([]byte("cat"))))

	fmt.Println("All entries:")
	for k, v := range s.All() {
		fmt.Printf("  %s: %s\n", k, v)
	}

	s.Pop([]byte("dog"))
	fmt.Println("After pop dog, len via iteration:")
	count := 0
	for range s.All() {
		count++
	}
	fmt.Println("count:", count)

	// Output:
	// cat -> purr
	// All entries:
	//   cat: purr
	//   dog: woof
	// After pop dog, len via iteration:
	// count: 1
}
