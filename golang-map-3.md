---
title: Golang map 实现（三）
date: 2020-04-26 09:21:23
tags:
  - golang
  - map
---

本文在golang map 数据结构的基础上，学习map 数据是如何访问的。

<!--more-->

## map 创建示例

在golang 中，访问 map 的方式有两种，例子如下：

```go

val := example1Map[key1]
val, ok := example1Map[key1]
```

第一种方式不判断是否存在key值，直接返回val （可能是空值）
第二种方式会返回一个bool 值，判断是否存在key 键值。（是不是和redis 的空值判断很类似）

## 那访问map 时，底层做了什么，我们一起来探究

对于不同的访问方式，会使用不同的方法，下面是内部提供的几种方法，我们一起来学习：

```go

// 迭代器中使用
func mapaccessK(t *maptype, h *hmap, key unsafe.Pointer) (unsafe.Pointer, unsafe.Pointer){}

// 不返回 bool
func mapaccess1(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {}
func mapaccess1_fat(t *maptype, h *hmap, key, zero unsafe.Pointer) unsafe.Pointer {}

// 返回 bool
func mapaccess2(t *maptype, h *hmap, key unsafe.Pointer) (unsafe.Pointer, bool) {}
func mapaccess2_fat(t *maptype, h *hmap, key, zero unsafe.Pointer) (unsafe.Pointer, bool) {}
```

这些方法有很大的相关性，下面我们逐一来学习吧。

### mapaccess1_fat, mapaccess2_fat

这两个方法，从字面上来看多了个fat，就是个宽数据。何以为宽，我们从下面代码找到原因：

```go
//src/cmd/compile/internal/gc/walk.go

if w := t.Elem().Width; w <= 1024 { // 1024 must match runtime/map.go:maxZero
  n = mkcall1(mapfn(mapaccess1[fast], t), types.NewPtr(t.Elem()), init, typename(t), map_, key)
} else {
  z := zeroaddr(w)
  n = mkcall1(mapfn("mapaccess1_fat", t), types.NewPtr(t.Elem()), init, typename(t), map_, key, z)
}
```

这是构建语法树时，mapaccess1 相关的代码（mapaccess2_fat 也类似）， 如果val 大于1024byte 的宽度，那会调用fat 后缀的方法。
原因是，在map.go 文件中，定义了val 0值的数组，代码如下：

```go
const maxZero = 1024 // must match value in cmd/compile/internal/gc/walk.go
var zeroVal [maxZero]byte
```

但是这个零值只能对宽度小于1024byte的宽度的数据有效，所以对于返回值（val）宽度小于1024 的，直接调用mapaccess1 方法即可，否则需要首先找一个对应的0值数据，然后调用mapaccess1_fat 方法，如果为0，传出对应的0值数据。

### mapaccess1， mapaccess2

mapaccess1 与 mapaccess2 的差别在于是否返回返回值，mapaccess2 将返回bool 类型作为是否不存在相应key的标识，mapaccess1 不会。所以，这里着重分析mapaccess2. 代码如下：

```go
func mapaccess2(t *maptype, h *hmap, key unsafe.Pointer) (unsafe.Pointer, bool) {
  // 竟态分析 && 内存扫描
  // ...

  if h == nil || h.count == 0 {
    // map 为空，或者size 为 0， 直接返回
  }
  if h.flags&hashWriting != 0 {
    // 这里会检查是否在写，如果在写直接panic
    throw("concurrent map read and map write")
  }

  // 拿到对应key 的hash，以及 bucket
  alg := t.key.alg
  hash := alg.hash(key, uintptr(h.hash0))
  m := bucketMask(h.B)
  b := (*bmap)(unsafe.Pointer(uintptr(h.buckets) + (hash&m)*uintptr(t.bucketsize)))
  if c := h.oldbuckets; c != nil {
    if !h.sameSizeGrow() {
      // There used to be half as many buckets; mask down one more power of two.
      m >>= 1
    }
    oldb := (*bmap)(unsafe.Pointer(uintptr(c) + (hash&m)*uintptr(t.bucketsize)))
    if !evacuated(oldb) {
      b = oldb
    }
  }

  // 获取tophash 值
  top := tophash(hash)
bucketloop:
  // 遍历解决冲突的链表
  for ; b != nil; b = b.overflow(t) {
    // 遍历每个bucket 上的kv
    for i := uintptr(0); i < bucketCnt; i++ {
      // 先匹配 tophash
      // ...

      // 获取k
      k := add(unsafe.Pointer(b), dataOffset+i*uintptr(t.keysize))
      if t.indirectkey() {
        k = *((*unsafe.Pointer)(k))
      }

      // 判断k是否相等,如果相等直接返回，否则继续遍历
      if alg.equal(key, k) {
        v := add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+i*uintptr(t.valuesize))
        if t.indirectvalue() {
          v = *((*unsafe.Pointer)(v))
        }
        return v, true
      }
    }
  }
  return unsafe.Pointer(&zeroVal[0]), false
}
```

访问map的流程比较简单：

- 首先，获取key 的hash值，并取到相应的bucket
- 其次，遍历对应的bucket，以及bucket 的链表（冲突链）
- 对于每个bucket 需要先匹配tophash 数组中的值，如果不匹配，则直接过滤。
- 如果hash 匹配成功，还是需要匹配key 是否相等，相等就返回，不等继续遍历。

这里需要注意一点：在tophash 数组中不仅会标识是否匹配hash值，还会标识下个数组中是否还有元素，减少匹配的次数。代码如下：

```go
if b.tophash[i] != top {
  if b.tophash[i] == emptyRest {
    break bucketloop
  }
  continue
}
```

tophash 的值有多种情况, 如果小于minTopHash，则作为标记使用。下面是标识含义:

```go
  // 标记为空，且后面没有数据了 (包括overflow 和 index)
  emptyRest      = 0 
  // 在被删除的时候设置为空
  emptyOne       = 1 
  // kv 数据被迁移到新hash表的 x 位置
  evacuatedX     = 2 
  // kv 数据被迁移到新hash表的 y 位置
  evacuatedY     = 3 
  // bucket 被转移走了，数据是空的
  evacuatedEmpty = 4 
  // 阈值标识
  minTopHash     = 5 
```


enptyRest, enptyOne 是有利于数据遍历的，减少了对数据的访问次数
evacuateX 和 evacuateY 与数据迁移有关，我们在赋值部分学习（赋值才有可能迁移）

## 总结

- map 中，val 如果宽度比较大，0值问题也需要多分配内存。所以，这种情况，使用指针肯定是合理的。（当然，内存拷贝也是一个问题）
- tophash 值的含义参考第一篇 bucket章节

今天的作业就交完了。下一篇将学习golang 赋值的实现。

## 参考

[1] [深入理解 Go map:初始化和访问](https://eddycjy.com/posts/go/map/2019-03-05-map-access/)

![](/images/weixin_logo.png)
