# go-skiplist

Simple skip list for Go.

Sorted `[]byte` key/value store with expected O(log n) operations.

## Install

```bash
go get github.com/toastsandwich/skiplist
```

## Usage

```go
import "github.com/toastsandwich/skiplist"

s := skiplist.NewSkipList(32, 0.5)

s.Put([]byte("user:1"), []byte("Alice"))
val, _ := s.Get([]byte("user:1"))

for k, v := range s.All() {
	// sorted order
}

s.Pop([]byte("user:1"))
```

## How Skip Lists Work

A skip list is a sorted linked list with extra "express lane" layers on top.

```
Level 2: HEAD ----------------------> [50] ------------> NIL
Level 1: HEAD ----------> [20] --> [50] --> [80] --> NIL
Level 0: HEAD --> [10] --> [20] --> [50] --> [80] --> NIL
```

- Every element lives on level 0.
- On insert, a node is randomly assigned a height. With the default p=0.5:
  - ~50% stay at level 0
  - ~25% reach level 1
  - ~12.5% reach level 2, and so on
- Search starts at the highest level, skips forward while keys are smaller, then drops down levels.
- Higher levels act as shortcuts, giving expected **O(log n)** performance for put, get, and delete.

## Benchmarks (heavy load)

Run on ~1 million elements (Intel Core i7-11800H):

```bash
go test -run=^$ -bench=. -benchmem -count=3 -benchtime=1s
```

| Op            | ns/op  | allocs/op |
|---------------|--------|-----------|
| Put           | 268    | 3         |
| Get           | 244    | 0         |
| Pop           | 572    | 4         |
| PutGetMix     | 619    | 3         |
| Iteration     | 6.3ms  | 0         |

- `Get`, `Pop`, and `PutGetMix` run against a preloaded list of 1M elements.
- `Iteration` = one full traversal over 500k elements.
- `Pop` benchmark does pop + re-insert to keep size stable.

`Get` has zero allocations on hits.

## License

MIT
