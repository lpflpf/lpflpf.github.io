---
title: Golang Map 实现 （四）
date: 2020-04-28 18:20:30
tags:
---

golang map 操作，是map 实现中较复杂的逻辑。因为当赋值时，为了减少hash 冲突链的长度过长问题，会做map 的扩容以及数据的迁移。而map 的扩容以及数据的迁移也是关注的重点。

<!--more-->

## 数据结构

首先，我们需要重新学习下map实现的数据结构：

```go
type hmap struct {
  count     int
  flags     uint8  
  B         uint8
  noverflow uint16
  hash0     uint32
  buckets    unsafe.Pointer
  oldbuckets unsafe.Pointer
  nevacuate  uintptr
  extra *mapextra
}

type mapextra struct {
  overflow    *[]*bmap
  oldoverflow *[]*bmap
  nextOverflow *bmap
}
```

hmap 是 map 实现的结构体。大部分字段在 第一节中已经学习过了。剩余的就是nevacuate 和extra 了。

首先需要了解搬迁的概念：当hash 中数据链太长，或者空的bucket 太多时，会操作数据搬迁，将数据挪到一个新的bucket 上，就的bucket数组成为了oldbuckets。bucket的搬迁不是一次就搬完的，是访问到对应的bucket时才可能会触发搬迁操作。（这一点是不是和redis 的扩容比较类似，将扩容放在多个访问上，减少了单次访问的延迟压力）

- nevactuate 标识的是搬迁的位置(也可以考虑为搬迁的进度）。标识目前 oldbuckets 中 （一个 array）bucket 搬迁到哪里了。
- extra 是一个map 的结构体，nextOverflow 标识的是申请的空的bucket，用于之后解决冲突时使用；overflow 和 oldoverflow 标识溢出的链表中正在使用的bucket 数据。old 和非old 的区别是，old 是为搬迁的数据。

理解了大概的数据结构，我们可以学习map的 赋值操作了。

## map 赋值操作

map 的赋值操作写法如下：

```go

   data := mapExample["hello"]

```

赋值的实现，golang 为了对不同类型k做了优化，下面时一些实现方法：

```go
func mapassign(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {}
func mapassign_fast32(t *maptype, h *hmap, key uint32) unsafe.Pointer {}
func mapassign_fast32ptr(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer {}
func mapassign_fast64(t *maptype, h *hmap, key uint64) unsafe.Pointer {}
func mapassign_fast64ptr(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer{}
func mapassign_faststr(t *maptype, h *hmap, s string) unsafe.Pointer {}
```

内容大同小异，我们主要学习mapassign 的实现。 

`mapassign` 方法的实现是查找一个空的bucket，把key赋值到bucket上，然后把val的地址返回,然后直接通过汇编做内存拷贝。
那我们一步步看是如何找空闲bucket的：

①  在查找key之前，会做异常检测，校验map是否未初始化，或正在并发写操作，如果存在，则抛出异常：（这就是为什么map 并发写回panic的原因）

```go
if h == nil {
  panic(plainError("assignment to entry in nil map"))
}
// 竟态检查 和 内存扫描

if h.flags&hashWriting != 0 {
  throw("concurrent map writes")
}
```

② 需要计算key 对应的hash 值，如果buckets 为空（初始化的时候小于一定长度的map 不会初始化数据）还需要初始化一个bucket

```go
alg := t.key.alg
hash := alg.hash(key, uintptr(h.hash0))

// 为什么需要在hash 后设置flags，因为 alg.hash可能会panic
h.flags ^= hashWriting

if h.buckets == nil {
  h.buckets = newobject(t.bucket) // newarray(t.bucket, 1)
}
```

③ 通过hash 值，获取对应的bucket。如果map 还在迁移数据，还需要在oldbuckets中找对应的bucket，并搬迁到新的bucket。

```go

// 通过hash 计算bucket的位置偏移
bucket := hash & bucketMask(h.B)

// 此处是搬迁逻辑，我们后续详解
if h.growing() {
  growWork(t, h, bucket)
}

// 计算对应的bucket 位置，和top hash 值
b := (*bmap)(unsafe.Pointer(uintptr(h.buckets) + bucket*uintptr(t.bucketsize)))
top := tophash(hash)
```

④ 拿到bucket之后，还需要按照链表方式一个一个查，找到对应的key， 可能是已经存在的key，也可能需要新增。

```go
for {
  for i := uintptr(0); i < bucketCnt; i++ {

    // 若 tophash 就不相等，那就取tophash 中的下一个
    if b.tophash[i] != top {

      // 若是个空位置，把kv的指针拿到。
      if isEmpty(b.tophash[i]) && inserti == nil {
        inserti = &b.tophash[i]
        insertk = add(unsafe.Pointer(b), dataOffset+i*uintptr(t.keysize))
        val = add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+i*uintptr(t.valuesize))
      }

      // 若后续无数据，那就不用再找坑了
      if b.tophash[i] == emptyRest {
        break bucketloop
      }
      continue
    }

    // 若tophash匹配时

    k := add(unsafe.Pointer(b), dataOffset+i*uintptr(t.keysize))
    if t.indirectkey() {
      k = *((*unsafe.Pointer)(k))
    }

    // 比较k不等，还需要继续找
    if !alg.equal(key, k) {
      continue
    }

    // 如果key 也相等，说明之前有数据，直接更新k，并拿到v的地址就可以了
    if t.needkeyupdate() {
      typedmemmove(t.key, k, key)
    }
    val = add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+i*uintptr(t.valuesize))
    goto done
  }
  // 取下一个overflow （链表指针）
  ovf := b.overflow(t)
  if ovf == nil {
    break
  }
  b = ovf
}
```

总结下这段程序，主要有几个部分：

a. map hash 不匹配的情况，会看是否是空kv 。如果调用了delete，会出现空kv的情况，那先把地址留下，如果后面也没找到对应的k（也就是说之前map 里面没有对应的Key），那就直接用空kv的位置即可。
b. 如果 map hash 是匹配的，需要判定key 的字面值是否匹配。如果不匹配，还需要查找。如果匹配了，那直接把key 更新（因为可能有引用），v的地址返回即可。
c. 如果上面都没有，那就看下一个bucket

⑤ 插入数据前，会先检查数据太多了，需要扩容，如果需要扩容，那就从第③开始拿到新的bucket，并查找对应的位置。

```go
if !h.growing() && (overLoadFactor(h.count+1, h.B) || tooManyOverflowBuckets(h.noverflow, h.B)) {
  hashGrow(t, h)
  goto again // Growing the table invalidates everything, so try again
}
```

⑥ 如果刚才看没有有空的位置，那就需要在链表后追加一个bucket，拿到kv。

```go
if inserti == nil {
  // all current buckets are full, allocate a new one.
  newb := h.newoverflow(t, b)
  inserti = &newb.tophash[0]
  insertk = add(unsafe.Pointer(newb), dataOffset)
  val = add(insertk, bucketCnt*uintptr(t.keysize))
}
```

⑦ 最后更新tophash 和 key 的字面值, 并解除hashWriting 约束

```go
// 如果非指针数据（也就是直接赋值的数据），还需要申请内存和拷贝
if t.indirectkey() {
  kmem := newobject(t.key)
  *(*unsafe.Pointer)(insertk) = kmem
  insertk = kmem
}
if t.indirectvalue() {
  vmem := newobject(t.elem)
  *(*unsafe.Pointer)(val) = vmem
}
// 更新tophash, k
typedmemmove(t.key, insertk, key)
*inserti = top

done:
if h.flags&hashWriting == 0 {
    throw("concurrent map writes")
  }
  h.flags &^= hashWriting
  if t.indirectvalue() {
    val = *((*unsafe.Pointer)(val))
  }
  return val
```

到这里，map的赋值基本就介绍完了。下面学习下步骤⑤中的map的扩容。

## Map 的扩容

有两种情况下，需要做扩容。一种是存的kv数据太多了，已经超过了当前map的负载。还有一种是overflow的bucket过多了。这个阈值是一个定值，经验得出的结论，所以我们这里不考究。

当满足条件后，将开始扩容。如果满足条件二，扩容后的buckets 的数量和原来是一样的，说明可能是空kv占据的坑太多了，通过map扩容做内存整理。如果是因为kv 量多导致map负载过高，那就扩一倍的量。

```go
func hashGrow(t *maptype, h *hmap) {
  bigger := uint8(1)
  // 如果是第二种情况，扩容大小为0
  if !overLoadFactor(h.count+1, h.B) {
    bigger = 0
    h.flags |= sameSizeGrow
  }
  oldbuckets := h.buckets

  // 申请一个大数组，作为新的buckets
  newbuckets, nextOverflow := makeBucketArray(t, h.B+bigger, nil)

  flags := h.flags &^ (iterator | oldIterator)
  if h.flags&iterator != 0 {
    flags |= oldIterator
  }
  
  // 然后重新赋值map的结构体，oldbuckets 被填充。之后将做搬迁操作
  h.B += bigger
  h.flags = flags
  h.oldbuckets = oldbuckets
  h.buckets = newbuckets
  h.nevacuate = 0
  h.noverflow = 0

  // extra 结构体做赋值
  if h.extra != nil && h.extra.overflow != nil {
    // Promote current overflow buckets to the old generation.
    if h.extra.oldoverflow != nil {
      throw("oldoverflow is not nil")
    }
    h.extra.oldoverflow = h.extra.overflow
    h.extra.overflow = nil
  }
  if nextOverflow != nil {
    if h.extra == nil {
      h.extra = new(mapextra)
    }
    h.extra.nextOverflow = nextOverflow
  }
}
```

总结下map的扩容操作。首先拿到扩容的大小，然后申请大数组，然后做些初始化的操作，把老的buckets，以及overflow做切换即可。

## map 数据的迁移

扩容完成后，需要做数据的迁移。数据的迁移不是一次完成的，是使用时才会做对应bucket的迁移。也就是逐步做到的数据迁移。下面我们来学习。

在数据赋值的第③步，会看需要操作的bucket是不是在旧的buckets里面，如果在就搬迁。下面是搬迁的具体操作：

```go
func growWork(t *maptype, h *hmap, bucket uintptr) {
  // 首先把需要操作的bucket 搬迁
  evacuate(t, h, bucket&h.oldbucketmask())
  
  // 再顺带搬迁一个bucket
  if h.growing() {
    evacuate(t, h, h.nevacuate)
  }
}
```

nevacuate 标识的是当前的进度，如果都搬迁完，应该和2^B的长度是一样的（这里说的B是oldbuckets 里面的B，毕竟新的buckets长度可能是2^(B+1))。

在evacuate 方法实现是把这个位置对应的bucket，以及其冲突链上的数据都转移到新的buckets上。

① 先要判断当前bucket是不是已经转移。 (oldbucket 标识需要搬迁的bucket 对应的位置)

```go
b := (*bmap)(add(h.oldbuckets, oldbucket*uintptr(t.bucketsize)))
// 判断
if !evacuated(b) {
  // 做转移操作
}
```

转移的判断直接通过tophash 就可以，判断tophash中第一个hash值即可 （tophash的作用可以参考第三讲）

```go
func evacuated(b *bmap) bool {
  h := b.tophash[0]
  // 这个区间的flag 均是已被转移
  return h > emptyOne && h < minTopHash
}
```

② 如果没有被转移，那就要迁移数据了。数据迁移时，可能是迁移到大小相同的buckets上，也可能迁移到2倍大的buckets上。这里xy 都是标记目标迁移位置的标记：x 标识的是迁移到相同的位置，y 标识的是迁移到2倍大的位置上。我们先看下目标位置的确定：

```go
var xy [2]evacDst
x := &xy[0]
x.b = (*bmap)(add(h.buckets, oldbucket*uintptr(t.bucketsize)))
x.k = add(unsafe.Pointer(x.b), dataOffset)
x.v = add(x.k, bucketCnt*uintptr(t.keysize))
if !h.sameSizeGrow() {
  // 如果是2倍的大小，就得算一次 y 的值
  y := &xy[1]
  y.b = (*bmap)(add(h.buckets, (oldbucket+newbit)*uintptr(t.bucketsize)))
  y.k = add(unsafe.Pointer(y.b), dataOffset)
  y.v = add(y.k, bucketCnt*uintptr(t.keysize))
}
```

③ 确定bucket位置后，需要按照kv 一条一条做迁移。（目的就是清除空闲的kv）

```go

// 遍历每个bucket
for ; b != nil; b = b.overflow(t) {
  k := add(unsafe.Pointer(b), dataOffset)
  v := add(k, bucketCnt*uintptr(t.keysize))

  // 遍历bucket 里面的每个kv
  for i := 0; i < bucketCnt; i, k, v = i+1, add(k, uintptr(t.keysize)), add(v, uintptr(t.valuesize)) {
    top := b.tophash[i]

    // 空的不做迁移
    if isEmpty(top) {
      b.tophash[i] = evacuatedEmpty
      continue
    }
    if top < minTopHash {
      throw("bad map state")
    }
    k2 := k
    if t.indirectkey() {
      k2 = *((*unsafe.Pointer)(k2))
    }
    var useY uint8
    if !h.sameSizeGrow() {
      // 2倍扩容的需要重新计算hash，
      hash := t.key.alg.hash(k2, uintptr(h.hash0))
      if h.flags&iterator != 0 && !t.reflexivekey() && !t.key.alg.equal(k2, k2) {
        useY = top & 1
        top = tophash(hash)
      } else {
        if hash&newbit != 0 {
          useY = 1
        }
      }
    }

    // 这些是固定值的校验，可以忽略
    if evacuatedX+1 != evacuatedY || evacuatedX^1 != evacuatedY {
      throw("bad evacuatedN")
    }

    // 设置oldbucket 的tophash 为已搬迁
    b.tophash[i] = evacuatedX + useY // evacuatedX + 1 == evacuatedY
    dst := &xy[useY]                 // evacuation destination
    if dst.i == bucketCnt {
      // 如果dst是bucket 里面的最后一个kv，则需要添加一个overflow
      dst.b = h.newoverflow(t, dst.b)
      dst.i = 0
      dst.k = add(unsafe.Pointer(dst.b), dataOffset)
      dst.v = add(dst.k, bucketCnt*uintptr(t.keysize))
    }
    // 填充tophash值， kv 数据
    dst.b.tophash[dst.i&(bucketCnt-1)] = top
    if t.indirectkey() {
      *(*unsafe.Pointer)(dst.k) = k2
    } else {
      typedmemmove(t.key, dst.k, k)
    }
    if t.indirectvalue() {
      *(*unsafe.Pointer)(dst.v) = *(*unsafe.Pointer)(v)
    } else {
      typedmemmove(t.elem, dst.v, v)
    }

    // 更新目标的bucket
    dst.i++
    dst.k = add(dst.k, uintptr(t.keysize))
    dst.v = add(dst.v, uintptr(t.valuesize))
  }
}
```

对于key 非间接使用的数据（即非指针数据），做内存回收

```go
if h.flags&oldIterator == 0 && t.bucket.kind&kindNoPointers == 0 {
  b := add(h.oldbuckets, oldbucket*uintptr(t.bucketsize))
  ptr := add(b, dataOffset)
  n := uintptr(t.bucketsize) - dataOffset

  // ptr 是kv的位置， 前面的topmap 保留，做迁移前的校验使用
  memclrHasPointers(ptr, n)
}
```

④ 如果当前搬迁的bucket 和 总体搬迁的bucket的位置是一样的，我们需要更新总体进度的标记 nevacuate

```go
// newbit 是oldbuckets 的长度，也是nevacuate 的重点
func advanceEvacuationMark(h *hmap, t *maptype, newbit uintptr) {
  // 首先更新标记
  h.nevacuate++

  // 最多查看2^10 个bucket
  stop := h.nevacuate + 1024
  if stop > newbit {
    stop = newbit
  }

  // 如果没有搬迁就停止了，等下次搬迁
  for h.nevacuate != stop && bucketEvacuated(t, h, h.nevacuate) {
    h.nevacuate++
  }

  // 如果都已经搬迁完了，oldbukets 完全搬迁成功，清空oldbuckets
  if h.nevacuate == newbit {
    h.oldbuckets = nil
    if h.extra != nil {
      h.extra.oldoverflow = nil
    }
    h.flags &^= sameSizeGrow
  }
}
```

## 总结

1. Map 的赋值难点在于数据的扩容和数据的搬迁操作。
2. bucket 搬迁是逐步进行的，每进行一次赋值，会做至少一次搬迁工作。
3. 扩容不是一定会新增空间，也有可能是只是做了内存整理。
4. tophash 的标志即可以判断是否为空，还会判断是否搬迁，以及搬迁的位置为X or Y。
5. delete map 中的key，有可能出现很多空的kv，会导致搬迁操作。如果可以避免，尽量避免。

![](/images/weixin_logo.png)
