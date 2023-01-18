---
title: golang 容器的学习与实践
date: 2020-05-03 10:42:54
tags:
  - golang
  - container
---

golang 提供了几个简单的容器供我们使用，本文在介绍几种Golang 容器的基础上，实现一个基于Golang 容器的LRU算法。

<!--more-->

## 容器介绍

Golang 容器位于 container 包下，提供了三种包供我们使用，heap、list、ring. 下面我们分别学习。

### heap

heap 是一个堆的实现。一个堆正常保证了获取/弹出最大（最小）元素的时间为log n、插入元素的时间为log n.
golang的堆实现接口如下：

```go
// src/container/heap.go
type Interface interface {
sort.Interface
Push(x interface{}) // add x as element Len()
Pop() interface{}   // remove and return element Len() - 1.
}
```

heap 是基于 sort.Interface 实现的。

```go
// src/sort/
type Interface interface {
// Len is the number of elements in the collection.
Len() int
// Less reports whether the element with
// index i should sort before the element with index j.
Less(i, j int) bool
// Swap swaps the elements with indexes i and j.
Swap(i, j int)
}
```

因此，如果要使用官方提供的heap，需要我们**实现如下几个接口**：

```go
Len() int {}              // 获取元素个数
Less(i, j int) bool  {}   // 比较方法
Swap(i, j int)            // 元素交换方法
Push(x interface{}){}     // 在末尾追加元素
Pop() interface{}         // 返回末尾元素
```

然后在使用时，我们可以**使用如下几种方法**：

```go
// 初始化一个堆
func Init(h Interface){}
// push一个元素倒堆中
func Push(h Interface, x interface{}){}
// pop 堆顶元素
func Pop(h Interface) interface{} {}
// 删除堆中某个元素，时间复杂度 log n
func Remove(h Interface, i int) interface{} {}
// 调整i位置的元素位置（位置I的数据变更后）
func Fix(h Interface, i int){}

```

### list 链表

list 实现了一个双向链表，链表不需要实现heap 类似的接口，可以直接使用。

链表的构造和使用：

```go
  // 返回一个链表对象
  func New() *List {}
  // 返回链表的长度
  func (l *List) Len() int {}
  // 返回链表中的第一个元素
  func (l *List) Front() *Element {}
  // 返回链表中的末尾元素
  func (l *List) Back() *Element {}
  // 移除链表中的某个元素
  func (l *List) Remove(e *Element) interface{} {}
  // 在表头插入值为 v 的元素
  func (l *List) PushFront(v interface{}) *Element {}
  // 在表尾插入值为 v 的元素
  func (l *List) PushBack(v interface{}) *Element {}
  // 在mark之前插入值为v 的元素
  func (l *List) InsertBefore(v interface{}, mark *Element) *Element {}
  // 在mark 之后插入值为 v 的元素
  func (l *List) InsertAfter(v interface{}, mark *Element) *Element {}
  // 移动e某个元素到表头
  func (l *List) MoveToFront(e *Element) {}
  // 移动e到队尾
  func (l *List) MoveToBack(e *Element) {}
  // 移动e到mark之前
  func (l *List) MoveBefore(e, mark *Element) {}
  // 移动e 到mark 之后
  func (l *List) MoveAfter(e, mark *Element) {}
  // 追加到队尾
  func (l *List) PushBackList(other *List) {}
  // 将链表list放在队列前
  func (l *List) PushFrontList(other *List) {}
```

我们可以通过 Value 方法访问 Element 中的元素。除此之外，我们还可以用下面方法做链表遍历：

```go
// 返回下一个元素
func (e *Element) Next() *Element {}
// 返回上一个元素
func (e *Element) Prev() *Element {}
```

队列的遍历：

```go
// l 为队列，
for e := l.Front(); e != nil; e = e.Next() {
  //通过 e.Value 做数据访问
}
```

### ring 循环列表

container 中的循环列表是采用链表实现的。

```go
// 构造一个包含N个元素的循环列表
func New(n int) *Ring {}
// 返回列表下一个元素
func (r *Ring) Next() *Ring {}
// 返回列表上一个元素
func (r *Ring) Prev() *Ring {}
// 移动n个元素 （可以前移，可以后移）
func (r *Ring) Move(n int) *Ring {}
// 把 s 链接到 r 后面。如果s 和r 在一个ring 里面，会把r到s的元素从ring 中删掉
func (r *Ring) Link(s *Ring) *Ring {}
// 删除n个元素 （内部就是ring 移动n个元素，然后调用Link)
func (r *Ring) Unlink(n int) *Ring {}
// 返回Ring 的长度，时间复杂度 n
func (r *Ring) Len() int {}
// 遍历Ring，执行 f 方法 （不建议内部修改ring）
func (r *Ring) Do(f func(interface{})) {}
```

访问Ring 中元素，直接 Ring.Value 即可。

## 容器的使用

LRU 算法 (Least Recently Used)，在做缓存置换时用的比较多。逐步淘汰最近未使用的cache，而使我们的缓存中持续保持着最近使用的数据。下面，我们通过map 和 官方包中的双向链表实现一个简单的lru 算法，用来熟悉golang 容器的使用。

```go
package main

import "fmt"
import "container/list"

// lru 中的数据
type Node struct {
  K, V interface{}
}

// 链表 + map
type LRU struct {
  list     *list.List
  cacheMap map[interface{}]*list.Element
  Size     int
}

// 初始化一个LRU
func NewLRU(cap int) *LRU {
  return &LRU{
    Size:     cap,
    list:     list.New(),
    cacheMap: make(map[interface{}]*list.Element, cap),
  }
}

// 获取LRU中数据
func (lru *LRU) Get(k interface{}) (v interface{}, ret bool) {
  // 如果存在，则把数据放到链表最前面
  if ele, ok := lru.cacheMap[k]; ok {
    lru.list.MoveToFront(ele)
    return ele.Value.(*Node).V, true
  }

  return nil, false
}

// 设置LRU中数据
func (lru *LRU) Set(k, v interface{}) {
  // 如果存在，则把数据放到最前面
  if ele, ok := lru.cacheMap[k]; ok {
    lru.list.MoveToFront(ele)
    ele.Value.(*Node).V = v  // 更新数据值
    return
  }
  
  // 如果数据是满的，先删除数据，后插入
  if lru.list.Len() == lru.Size {
    last := lru.list.Back()
    node := last.Value.(*Node)
    delete(lru.cacheMap, node.K)
    lru.list.Remove(last)
  }

  ele := lru.list.PushFront(&Node{K: k, V: v})
  lru.cacheMap[k] = ele
}
```

## 其他

1. 上述的容器都不是goroutines 安全的
2. 上面的lr 也不是goroutines 安全的
3. Ring 中不建议在Do 方法中修改Ring 的指针，行为是未定义的

