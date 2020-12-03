golang-lru
==========

This provides the `lru` package which implements a variable cost
thread safe LRU cache. It is based on the cache in Groupcache.

Documentation
=============

Full docs are available on [Godoc](http://godoc.org/github.com/hashicorp/golang-lru)

Example
=======

Using the LRU is very simple:

```go
l, _ := New(128)
for i := 0; i < 256; i++ {
    l.Add(i, nil, 2)
}
if l.Len() != 64 {
    panic(fmt.Sprintf("bad len: %v", l.Len()))
}
```
