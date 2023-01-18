---
title: golang map实现（一）
subtitle: golang map的数据结构
date: 2020-04-21 16:56:17
tags:
  - golang
  - map
---

本文学习 Golang 的 Map 数据结构，以及map buckets 的数据组织结构。

<!--more-->

# hash 表是什么

从大学的课本里面，我们学到：hash 表其实就是将key 通过hash算法映射到数组的某个位置,然后把对应的val存放起来。
如果出现了hash冲突（也就是说，不同的key被映射到了相同的位置上时），就需要解决hash冲突。解决hash冲突的方法还是比较多的，比如说开放定址法，再哈希法，链地址法，公共溢出区等(复习下大学的基本知识)。

其中链地址法比较常见，下面是一个链地址法的常见模式：

![hash_common](golang_hash_1_common.jpg)

Position 指通过Key 计算出的数组偏移量。例如当 Position = 6 的位置已经填满KV后，再次插入一条相同Position的数据将通过链表的方式插入到该条位置之后。

在php的Array 中是这么实现的，golang中也基本是这么实现。下面我们学习下Golang中map的实现。

## Golang Map 实现的数据结构

Golang的map中，首先把kv 分在了N个桶中，每个桶中的数据有8条（bucketCnt）。如果一个桶满了(overflow)，也会采用链地址法解决hash 的冲突。

下面是定义一个hashmap的结构体：

```go
type hmap struct {
  // 长度
  count     int
  // map 的标识, 下方做了定义
  flags     uint8  
  // 实际buckets 的长度为 2 ^ B
  B         uint8
  // 从bucket中溢出的数量，（存在extra 里面)
  noverflow uint16
  // hash 种子，做key 哈希的时候会用到
  hash0     uint32
  // 存储 buckets 的地方
  buckets    unsafe.Pointer
  // 迁移时oldbuckets中存放部分buckets 的数据
  oldbuckets unsafe.Pointer
  // 迁移的数量
  nevacuate  uintptr
   // 一些额外的字段，在做溢出处理以及数据增长的时候会用到
  extra *mapextra
}
const (
  iterator     = 1 // 有一个迭代器在使用buckets
  oldIterator  = 2 // 有一个迭代器在使用oldbuckets
  hashWriting  = 4 // 并发写，通过这个标识报panic
  sameSizeGrow = 8 // the current map growth is to a new map of the same size
)
type mapextra struct {
  overflow    *[]*bmap
  oldoverflow *[]*bmap
  nextOverflow *bmap
}
type bmap struct {
  tophash [bucketCnt]uint8
}
```

表中除了对基本的hash数据结构做了定义外，还对数据迁移、扩容等操作做了定义，这里我们可以忽略，等学习到时我们再深入了解。

## 深入 桶列表 (buckets)

buckets 字段中是存储桶数据的地方。正常会一次申请至少2^N长度的数组，数组中每个元素就是一个桶。N 就是结构体中的B。这里面要注意以下几点：

1. **为啥是2的幂次方**  为了做完hash后，通过掩码的方式取到数组的偏移量, 省掉了不必要的计算。
2. **B 这个数是怎么确定的**  这个和我们map中要存放的数据量是有很大关系的。我们在创建map的时候来详述。
3. **bucket 的偏移是怎么计算的** hash 方法有多个，在 runtime/alg.go 里面定义了。不同的类型用不同的hash算法。算出来是一个uint32的一个hash 码，通过和B取掩码，就找到了bucket的偏移了。下面是取对应bucket的例子：

```go
// 根据key的类型取相应的hash算法
alg := t.key.alg
hash := alg.hash(key, uintptr(h.hash0))
// 根据B拿到一个掩码
m := bucketMask(h.B)
// 通过掩码以及hash指，计算偏移得到一个bucket
b := (*bmap)(add(h.buckets, (hash&m)*uintptr(t.bucketsize)))
```

## 深入 桶 (bucket)

一个桶的示意图如下：
![golang_map_1_hash_bucket](golang_hash_1_bucket.jpg)

每个桶里面，可以放8个k，8个v，还有一个overflow指针（就是上面的next），用来指向下一个bucket 的地址。在每个bucket的头部，还会放置一个tophash，也就是bmap 结构体。这个数组里面存放的是key的hash值，用来对比我们key生成的hash和存出的hash是否一致（当然除了这个还有其他的用途，后面讲数据访问的时候会讲到）。 tophash中的数据，是从计算的hash值里面截取的。获取bucket 是用的低bit位的hash，tophash 使用的是高bit位的hash值（8位）

1. **为啥bucket 一次要存8个kv，而不是一个kv放一个bucket，然后链地址法做处理就OK了** 据我分析，有几点原因: a， 一次分配8个kv的空间，可以减少内存的分配频次; b，减少了overflow指针的内存占用，比如说8个kv，采用一个一个存储的话，需要8 * 8B （64位机） = 64B的数据存下一个的地址，而采用go实现的这种方式，只需要 8B + 8B (bmap的大小） = 16B 的数据就可以了。
2. **为啥需要用tophash** 一般的hash 实现逻辑是直接和key比较，如果比较成功，这找到相应key的数据。但是这里用到了tophash，好处是可以减少key的比较成本（毕竟key 不一定都是整数形式存在的）
3. **为啥是8个** 8 * 8B = 64B 整好是64位机的一个最小寻址空间，不过可以通过修改源码自定义吧。
4. **为什么key 和val 要分开放** 这个也比较好理解，key 和val 都是用户可以自定义的。如果key是定长的（比如是数字，或者 指针之类的，大概率是这样。）内存是比较整齐的，利于寻址吧。

## 技术总结

golang 实现的map比朴素的hashmap 在很多方面都有优化。

1. 使用掩码方式获取偏移，减少判断。
2. bucket 存储方式的优化。
3. 通过tophash 先进行一次比较，减少key 比较的成本。
4. 当然，有一点是不太明白的，为啥 overflow 指针要放在 kv 后面？ 放在tophash 之后的位置岂不是更完美？

今天的作业就交完了。下一篇将学习golang map的数据初始化实现。

## 参考

[1] [深入理解 Go map:初始化和访问](https://eddycjy.com/posts/go/map/2019-03-05-map-access/)

![](/images/weixin_logo.png)
