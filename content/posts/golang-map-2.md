---
title: 	Golang Map 实现（二）
date: 2020-04-23 17:45:23
tags:
  - golang
  - map
---

本文在golang map 数据结构的基础上，学习一个make 是如何构造的。

<!--more-->

## map 创建示例

在golang 中，初始化一个map 算是有两种方式。

```go
example1Map := make(map[int64]string)
example2Map := make(map[int64]string, 100)
```

第一种方式默认不指定map的容量，第二种会指定后续map的容量估计为100，希望在创建的时候把空间就分配好。

## 当make创建map时，底层做了什么

对于不同的初始化方式，会使用不同的方式。下面是提供的几种初始化方法：

```go
// hint 就是 make 初始化map 的第二个参数
func makemap(t *maptype, hint int, h *hmap) *hmap
func makemap64(t *maptype, hint int64, h *hmap) *hmap
func makemap_small() *hmap
```

区别在于：
如果不指定 hint，就调用makemap_small；
如果make 第二个参数为int64, 则调用makemap64；
其他情况调用makemap方法。下面我们逐一学习。

### makemap_small

```go
func makemap_small() *hmap {  
  h := new(hmap)
  h.hash0 = fastrand()
  return h
}
```

fastrand 是创建一个seed，在生成hash值时使用。
所以在makemap_small 时，只是创建了一个hmap 的结构体，并没有初始化buckets.

### makemap64

```go
func makemap64(t *maptype, hint int64, h *hmap) *hmap {
  if int64(int(hint)) != hint {
    hint = 0
  }
  return makemap(t, int(hint), h)
}
```

makemap64 是对于传入的第二个参数为int64 的变量使用的。 如果hint的值大于int最大值，就将hint赋值为0，否则和makemap 初始化没有差别。为什么不把大于2^31 - 1 的map 直接初始化呢？因为在hmap 中 count 的值就是int，也就是说**map最大就是 2^31 - 1 的大小**。

### makemap

这个是初始化map的核心代码了，需要我们慢慢品味。

一开始，我们需要了解下maptype这个结构， maptype 标识一个map 数据类型的定义，当然还有其他的类型，比如说interfacetype，slicetype，chantype 等。maptype 的定义如下：

```go
type maptype struct {
  typ        _type  // type 类型
  key        *_type // key 的type
  elem       *_type // value 的type
  bucket     *_type // internal type representing a hash bucket
  keysize    uint8  // key 的大小
  valuesize  uint8  // value 的大小
  bucketsize uint16 // size of bucket
  flags      uint32
}
```

maptype 里面存储了kv的对象类型，bucket类型，以及kv占用内存的大小。以及bucketsize的大小，还有一些标记字段（flags）。在map 实现时，需要用到这些字段做偏移计算等。

下面是 makemap 的代码：

```go

// hint 需要创建的 map 大小(预计要添加多少元素)
func makemap(t *maptype, hint int, h *hmap) *hmap {
  mem, overflow := math.MulUintptr(uintptr(hint), t.bucket.size)
  if overflow || mem > maxAlloc {
    hint = 0
  }

  // initialize Hmap
  if h == nil {
    h = new(hmap)
  }

  // xorshift64+ 算法, 可以研究下
  h.hash0 = fastrand()

  // 计算B 的值
  // 如果大于8，就先申请好。
  // 申请规则就是刚好满足 hint < 6.5 * 2 ^ B 的时候 （B 最大是63）
  // 其中6.5 相当于每个bucket 链表中，平均有6.5个bucket
  // 所以最长的map，应该是 6.5 * 2^63 (正常用肯定不会溢出)
  
  B := uint8(0)
  for overLoadFactor(hint, B) {
    B++
  }
  h.B = B

  // 接着数据初始化, 如果 容量小于等于8的，就在用的时候初始化, B 为0
  if h.B != 0 {
    var nextOverflow *bmap
    // 申请一个buckets 数组
    h.buckets, nextOverflow = makeBucketArray(t, h.B, nil)
    if nextOverflow != nil {
      h.extra = new(mapextra)
      h.extra.nextOverflow = nextOverflow
    }
  }

  return h
}
```

首先，通过bucketsize 和hint 的值，计算出需要分配的内存大小mem， 以及是否会overflow （大于指针的最大地址范围），如果溢出或者申请的内存大于最大可以申请的内存时，就设置hint为0了，直接不初始化buckets了。

接着，和makemap_small 一样，初始化一个随机的种子。

然后，计算B的值. 在overLoadfactor 中，判断了hint 的大小。如果小于等于8，那B就不再赋值，直接不初始化数据。如果B大于8，那就计算B了。这里涉及到一个填充因子的概念。大概意思就是说，每个hash值（也就是pos）中，平均放多少个kv数据，默认是6.5；所以判断标准就是hint 必须满足如下的条件：

```go
hint < 6.5 * (1 << B)
```

通过增加B的值，直到上面的表达式满足为止。这样B就初始化好了。

最后，申请一个bucket数组，赋值给buckets，如果有多申请出来的buckets，那就赋值给extra.nextOverflow, 当溢出之后，从多申请出来的buckets 里面取（也是为了避免内存分配）。

下面就详细看下初始化一个buckets的构建。

## makeBucketArray

makeBucketArray 用于初始化一个Bucket 数组。也就是hmap 中的buckets，下面是相关代码：

```go
func makeBucketArray(t *maptype, b uint8, dirtyalloc unsafe.Pointer) 
	(buckets unsafe.Pointer, nextOverflow *bmap) {
  base := bucketShift(b)
  nbuckets := base
  // 为了防止溢出的迁移，加一点冗余的bucket
  if b >= 4 {  
    // ... 修改nbuckets
  }

  // 如果之前没有分配过，那直接分配
  if dirtyalloc == nil {
    buckets = newarray(t.bucket, int(nbuckets))
  } else {
    // 使用以前分配好的
    buckets = dirtyalloc
    size := t.bucket.size * nbuckets
    if t.bucket.kind&kindNoPointers == 0 {
      memclrHasPointers(buckets, size)
    } else {
      memclrNoHeapPointers(buckets, size)
    }
  }

  if base != nbuckets {
	// 处理多申请出来的bucket
  }
  return buckets, nextOverflow
}
```

这里用到了比较多的指针计算，需要细细品读。

- 首先，就是就是通过B计算一个base值，base = 1 << B （2 ^ B)
nbuckets 是需要申请的数组的长度，正常情况下 base 值就是数组长度。但是，如果 base 大于16时，会预分配一些需要后期做overflow的bucket。这个overflow的计算规则如下：

```go
    nbuckets += bucketShift(b - 4)
    sz := t.bucket.size * nbuckets
    up := roundupsize(sz)
    if up != sz {
      nbuckets = up / t.bucket.size
    }
```

在base 的基础上，多分配 base / 16 长度的bucket。然后根据内存的分配规则（包括了页大小和内存对齐等规则），计算出合适的分配内存的大小，然后计算出 bucket 的分配个数 nbuckets.

- 其次，如果有之前未分配内存，那就初始化一个数组（终于等到了这一步），如过有dirtyalloc， 那就使用dirtyalloc 的内存（其实是用来清除map中数据使用的），然后把dirtyalloc中不需要的数据清除引用。

- 最后，如果除了需要申请的base 长度的bucket外，还多申请了一些bucket，下面是对多申请的数据做的处理：

```go
    // 上面添加了一些nbuckets 防止溢出，所以B 值取模就不太合理了，所以有一个mapextra 的数据节点
    // 数据分配也很有趣，从刚申请的buckets数组中，取出后面的一段分给mapextra
    // nextOverflow 分配给mapextra
    nextOverflow = (*bmap)(add(buckets, base*uintptr(t.bucketsize)))

    // 取nextOverflow 里面的最后一个元素
    // 并把最后一个buckets 的末尾偏移的指针指向空闲的bucket (目前就是第一个buckets 了)
    last := (*bmap)(add(buckets, (nbuckets-1)*uintptr(t.bucketsize)))
    last.setoverflow(t, (*bmap)(buckets))
```

先计算出多申请出来的内存地址 nextOverflow，然后计算出 申请的最后一块bucket的地址，然后将最后一块bucket的overflow指针（指向链表的指针）指向buckets 的首部。 原因呢，是为了将来判断是否还有空的bucket 可以让溢出的bucket空间使用。


今天的作业就交完了。下一篇将学习golang map的数据初始化实现。

## 参考

[1] [深入理解 Go map:初始化和访问](https://eddycjy.com/posts/go/map/2019-03-05-map-access/)

![](/images/weixin_logo.png)

