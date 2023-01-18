---
title: golang sync.Map 实现
date: 2020-11-21 13:55:40
tags:
  - golang
  - sync

category: golang
---

本文主要简单介绍 sync.Map 的使用和实现。

<!--more-->

众所周知，Golang 的map是非协程安全的（并发读写数据是不允许的，但是可以并发读）。因此，在 Golang1.9 的版本中加入了 sync.Map 包，用于并发的访问 map。

下面我们简单学习sync.Map 的使用和实现。

### 如何使用

sync.Map 和map 在使用上有较大区别。map 为内置类型，sync.Map 实质是实现了一个带有一些操作方法的Struct 对象。因此，在使用sync.Map包做数据存取时，其实是调用了对象的一些方法来实现的。下面是sync.Map使用的一个简单例子，包含了大部分日常所需方法。

```go
    var cache sync.Map
	var k1 = "key1"
	var v1 = []int{1, 2, 3}

	println("==>Store, Load<==")
	cache.Store(k1, v1)

	if val, ok := cache.Load(k1); ok {
		fmt.Println(val.([]int))
	}

	println("==>LoadOrStore<==")
	k2 := "key2"
	v2 := "val2"

	fmt.Println(cache.LoadOrStore(k2, v2))
	fmt.Println(cache.LoadOrStore(k2, v2))

	println("==>Range<==")
	cache.Range(func(k, v interface{}) bool {
		fmt.Println(k, v)
		return true
	})

	println("==>Delete k2<==")
	cache.Delete(k2)
	fmt.Println(cache.Load(k2))
```

> 这里需要注意的是，在sync.Map中，没有提供 len 方法。

### sync.Map 实现

为了更好的理解sync.Map，有必要学习 sync.Map 是如何实现的。数据结构如下：

```go 
type Map struct {
	mu Mutex // 锁map的互斥锁
	read atomic.Value // 读结构
	dirty map[interface{}]*entry
	misses int // 统计
}

type readOnly struct {
	m       map[interface{}]*entry
	amended bool // 标记 dirty 中存在 read 里不存在的key
}

type entry struct {
    p unsafe.Pointer
}

var expunged = unsafe.Pointer(new(interface{}))
```

- `mu` 在数据访问上，通过互斥锁mu来保证 dirty 做读写操作的互不冲突。

- `read` 对象通过原子访问的方式保存,保存对象为 readOnly 类型，包含了存储数据的 m, 和一个标识 amended。在大部分情况下，读 read 不需要加锁访问。这里可以减少抢锁带来的消耗。amended 标识是否在 dirty 中有 read 中没有的数据。

- `dirty` 也保存了一个 map 数据。当用户写操作时，如果 read 中没有对应的 key，就会加锁把数据写入 dirty 中。dirty 如果不为 nil 的情况下，read 的数据应该是 dirty 数据的子集。

- `misses` 为一个统计参数，在数据访问时，如果 key 在 read 中不存在, 且 amended 标识为 true， dirty 增加 1， 当 misses 值大于 dirty 中元素的大小时，将 dirty 中的数据替换为 read 的 map。

- `*entry.p` 保存了value值的指针。p 存在多种情况:
  - p 为 `expunged`, 标记在 read 中将删除的数据。在下一次 dirty 转为 read 数据时将被删除。
  - p 为 nil，标识删除，但是 key 位置还保留，在 dirtyLocked 中，被转为 expunged
  - 数据存在，在调用 Delete 方法时，将被置为 nil


通过简单的描述，可以理解为读操作，尽量访问 read。写操作大部分情况下会访问 dirty （除非是做覆盖操作）。

下面，我们对比较常用的几个操作做细致分析。

#### Load

Load 操作从Map中获取 key 对应的value 值。

```go
func (m *Map) Load(key interface{}) (value interface{}, ok bool) {
    // 读操作首先从read中读取，如果读到数据，则返回
	read, _ := m.read.Load().(readOnly)
	e, ok := read.m[key]
	if !ok && read.amended {
        // 如果没有读到数据，并且存在dirty中有，read中没有的数据
		m.mu.Lock()
		read, _ = m.read.Load().(readOnly)
		e, ok = read.m[key]
		if !ok && read.amended {
			e, ok = m.dirty[key]
            // 出现了从read取失败的情况，miss += 1, 并考虑是否需要将dirty数据搬迁至read
			m.missLocked()
		}
		m.mu.Unlock()
	}
	if !ok {
		return nil, false
	}
	return e.load()
}


func (m *Map) missLocked() {
	m.misses++
    // misses 过多的话，就做一次copy
	if m.misses < len(m.dirty) {  
		return
	}
	m.read.Store(readOnly{m: m.dirty})
	m.dirty = nil
	m.misses = 0
}

```

Load 方法比较好理解。从read中取数据。在未取到并且存在脏数据的情况下，到dirty中取。如果miss过多的话，就把dirty放到read中。
Load中，如果在read中出现了，就不需要加锁。反之需要加锁。这里也可以看出，在读的频次远远大于写时，大部分情况下是不需要加锁的，这也是sync.Map 的优势。


#### Store

```go
func (m *Map) Store(key, value interface{}) {
	// 首先看read中是否存在对应的key，如果存在，直接替换即可
	read, _ := m.read.Load().(readOnly)
	if e, ok := read.m[key]; ok && e.tryStore(&value) {
		return
	}

	m.mu.Lock()
	read, _ = m.read.Load().(readOnly)
	if e, ok := read.m[key]; ok {
		// 如果之前是删除状态
		if e.unexpungeLocked() { 
			m.dirty[key] = e
		}
		e.storeLocked(&value)
	} else if e, ok := m.dirty[key]; ok {  
		// 如果 read 中不存在， dirty 中存在，那和store 一样，保存即可
		e.storeLocked(&value)
	} else {  
		// read/dirty 中都不存在
		if !read.amended {
			// 构造一个dirty 数据集 （包含目前 read 中的非删除的数据）  O(n)
			m.dirtyLocked()  
			// 设置 amended 与 dirty 不一致
			m.read.Store(readOnly{m: read.m, amended: true})  
		}
		m.dirty[key] = newEntry(value) // 数据保存至dirty
	}
	m.mu.Unlock()
}

// 清理read.m中的数据
func (m *Map) dirtyLocked() {
	if m.dirty != nil {
		return
	}

	read, _ := m.read.Load().(readOnly)
	m.dirty = make(map[interface{}]*entry, len(read.m))
	for k, e := range read.m {   // 真正删除数据
		if !e.tryExpungeLocked() {
			m.dirty[k] = e
		}
	}
}
```

Store 方法略微复杂，在read中存在对应key时，直接替换即可，此处也不需要加锁。
如果不存在，就需要加锁新增key 了。 如果dirty中存在，则直接保存。如果都不存在，则首先判断是否有修正过的数据，如果没有，需要调用dirtyLocked 方法，将read 方法中的未删除的数据copy 到新创建的map中，并标记read中nil值为 expunged（如果被标记过，那该value在read中不能被修改了）。重新设置amended 为 被修改的数据，并将新增的kv赋值到dirty中。

需要注意的是，在调用tryStore 更新 read 中的value值时，需要判断是否p 被标记为 unexpunged. 如果被标记为 unexpunged，则不能被更新。原因是: 在标记 unexpunged 后，在 dirty 中将不存在该值。如果做了更新，read中的数据将在dirty中不存在，导致在未来dirty迁移为read.m 时出现数据丢失。

#### Delete

```go
func (m *Map) LoadAndDelete(key interface{}) (value interface{}, loaded bool) {
	read, _ := m.read.Load().(readOnly)
	e, ok := read.m[key]

	// 删除操作大部分情况下也只是做nil标记，并不是直接删除
	if !ok && read.amended { // 如果read 里面没有，并且有脏数据的时候，就需要检查 dirty 的数据
		m.mu.Lock()
		read, _ = m.read.Load().(readOnly)
		e, ok = read.m[key]
		if !ok && read.amended {
			e, ok = m.dirty[key]
			m.missLocked()  // 可能需要将 dirty -> read
		}
		m.mu.Unlock()
	}
	if ok {
		return e.delete() // 标记 e.p = nil
	}
	return nil, false
}

// Delete deletes the value for a key.
func (m *Map) Delete(key interface{}) {
	m.LoadAndDelete(key)
}
```

 - 对于在read中存在的数据，删除操作，只是通过标记value值为nil，并不会实际删除对应key. 这种情况下，不需要加锁。


### 技术总结

1. 为什么在store中，read 可以不加锁修改map值。
正常情况下，对map的赋值是需要加锁的(不然可能会出现panic)，但是，为了减少加锁的消耗，map中存储的是entry对象，对象中保存的entry。赋值只与entry.p 相关，与map的赋值没有关系了。
2. 理论上，在read不存在key 的情况下，删除dirty中的key，只需要直接删除key 即可。（看github.com中源码已修改）
3. 为了保证操作的原子性，在做entry中数据的赋值时，均采用 atomic.StorePointer, atomic.LoadPointer, atomic.CompareAndSwapPointer 保证了赋值的有序和原子性。
4. 为了保证无锁状态下读取 read.m，read.m 对象是只读的，无法增加和删除key。通过标记p的值来删除，以及通过使用 dirty 替换 read.m 做数据的更新。
5. 从源码角度看来性能：
  - 不存在删除的情况，且key值比较固定，大部分情况是不需要加锁的。
  - 对于读多写少的情况，大部分情况也是从read中取值，不需要加锁的。
  - 对于存取的kv数据量非常大（百万级别）的情况下，一次 Store 可能需要对所有的read做遍历，并将未标记删除的entry赋值给dirty,这时可能出现卡顿现象。

对于协程安全的map，除了可以使用sync.Map 对象外，还可以用map +读写锁的方式简单暴力的实现协程安全的map。另外`github.com/orcaman/concurrent-map` 包也是一个不错的实现。concurrent-map 采用分段锁的方式实现了一个协程安全的map，在某些场景下比 sync.Map 有很大优势。具体如何选取，我们在接下来的文章中做具体分析。


