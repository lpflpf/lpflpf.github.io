---
title: 消息队列 NSQ 源码学习笔记 (二)
date: 2019-09-04 11:01:39
tags:
  - nsq
  - golang
  - 消息队列
  - diskqueue
---

> NSQ 消息队列实现消息落地使用的是 FIFO 队列。
> 实现为 **diskqueue** , 使用包 `github.com/nsqio/go-diskqueue` ,本文主要对 `diskqueue`的实现做介绍。

<!--more-->

## 功能定位

- 在NSQ 中， diskqueue 是一个实例化的 BackendQueue, 用于**保存在内存中放不下的消息**。使用场景如Topic 队列中的消息，Channel 队列中的消息
- 实现的功能是一个FIFO的队列，实现如下功能:
  - **支持消息的插入、清空、删除、关闭操作**
  - **可以返回队列的长度**(写和读偏移的距离)
  - 具有读写功能，FIFO 的队列

## diskqueue 的实现

  BackendQueue 接口如下：

```go
type BackendQueue interface {
    Put([]byte) error      // 将一条消息插入到队列中
    ReadChan() chan []byte // 返回一个无缓冲的chan
    Close() error          // 队列关闭
    Delete() error         // 删除队列 （实际在实现时，数据仍保留）
    Depth() int64          // 返回读延迟的消息量
    Empty() error          // 清空消息 （实际会删除所有的记录文件）
}
```

### 数据结构

对于需要原子操作的64bit 的字段，需要放在struct 的最前面，原因请看学习总结第一条
数据结构中定义了 文件的读写位置、一些文件读写的控制变量，以及相关操作的channel.

```go
// diskQueue implements a filesystem backed FIFO queue
type diskQueue struct {
    // 64bit atomic vars need to be first for proper alignment on 32bit platforms

    // run-time state (also persisted to disk)
    readPos      int64               // 读的位置
    writePos     int64               // 写的位置
    readFileNum  int64               // 读文件的编号
    writeFileNum int64               // 写文件的编号
    depth        int64               // 读写文件的距离 (用于标识队列的长度)

    sync.RWMutex

    // instantiation time metadata
    name            string           // 标识队列名称，用于落地文件名的前缀 
    dataPath        string           // 落地文件的路径
    maxBytesPerFile int64            // 每个文件最大字节数
    minMsgSize      int32            // 单条消息的最小大小
    maxMsgSize      int32            // 单挑消息的最大大小
    syncEvery       int64            // 每写多少次刷盘一次
    syncTimeout     time.Duration    // 至少多久会刷盘一次
    exitFlag        int32            // 退出标识
    needSync        bool             // 如果 needSync 为true， 则需要fsync刷新metadata 数据

    // keeps track of the position where we have read
    // (but not yet sent over readChan)
    nextReadPos     int64            // 下一次读的位置
    nextReadFileNum int64            // 下一次读的文件number

    readFile  *os.File               // 读 fd
    writeFile *os.File               // 写 fd
    reader    *bufio.Reader          // 读 buffer
    writeBuf  bytes.Buffer           // 写 buffer

    // exposed via ReadChan()
    readChan chan []byte             // 读channel

    // internal channels
    writeChan         chan []byte    // 写 channel
    writeResponseChan chan error     // 同步写完之后的 response
    emptyChan         chan int       // 清空文件的channel
    emptyResponseChan chan error     // 同步清空文件后的channel
    exitChan          chan int       // 退出channel
    exitSyncChan      chan int       // 退出命令同步等待channel

    logf AppLogFunc                  // 日志句柄
}
```

### 初始化一个队列

初始化一个队列，需要定义前缀名， 数据路径，每个文件的最大字节数，消息最大最小限制，以及刷盘频次和最长刷盘时间，最后还有一个日志函数

```go
func New(name string, dataPath string, maxBytesPerFile int64,
    minMsgSize int32, maxMsgSize int32,
    syncEvery int64, syncTimeout time.Duration, logf AppLogFunc) Interface {
    d := diskQueue{
        name:              name,
        dataPath:          dataPath,
        maxBytesPerFile:   maxBytesPerFile,
        minMsgSize:        minMsgSize,
        maxMsgSize:        maxMsgSize,
        readChan:          make(chan []byte),
        writeChan:         make(chan []byte),
        writeResponseChan: make(chan error),
        emptyChan:         make(chan int),
        emptyResponseChan: make(chan error),
        exitChan:          make(chan int),
        exitSyncChan:      make(chan int),
        syncEvery:         syncEvery,
        syncTimeout:       syncTimeout,
        logf:              logf,
    }

    // no need to lock here, nothing else could possibly be touching this instance
    err := d.retrieveMetaData()
    if err != nil && !os.IsNotExist(err) {
        d.logf(ERROR, "DISKQUEUE(%s) failed to retrieveMetaData - %s", d.name, err)
    }

    go d.ioLoop()
    return &d
}
```

可以看出, 队列中均使用不带cache 的chan，消息只能阻塞处理。

`d.retrieveMetaData()` 是从文件中恢复元数据。

`d.ioLoop()` 是队列的事件处理逻辑，后文详细解答

### 消息的读写

#### 文件格式

文件名 `"name" + .diskqueue.%06d.dat` 其中， name 是 topic, 或者topic + channel 命名.
数据采用二进制方式存储， 消息大小+ body 的形式存储。

#### 消息读操作

- 如果readFile 文件描述符未初始化， 则需要先打开相应的文件，将偏移seek到相应位置，并初始化reader buffer
- 初始化后，首先读取文件的大小， 4个字节，然后通过文件大小获取相应的body 数据
- 更改相应的偏移。如果偏移达到文件最大值，则会关闭相应文件，读的文件编号 + 1

#### 消息写操作

- 如果writeFile 文件描述符未初始化，则需要先打开相应的文件，将偏移seek到文件末尾。
- 验证消息的大小是否符合要求
- 将body 的大小和body 写入 buffer 中，并落地
- depth +1，
- 如果文件大小大于每个文件的最大大小，则关闭当前文件，并将写文件的编号 + 1

### 事件循环 ioLoop

ioLoop 函数，做所有时间处理的操作,包括：

- 消息读取
- 写操作
- 清空队列数据
- 定时刷新的事件

```go
func (d *diskQueue) ioLoop() {
    var dataRead []byte
    var err error
    var count int64
    var r chan []byte

    // 定时器的设置
    syncTicker := time.NewTicker(d.syncTimeout)

    for {
        // 若到达刷盘频次，标记等待刷盘
        if count == d.syncEvery {
            d.needSync = true
        }

        if d.needSync {
            err = d.sync()
            if err != nil {
                d.logf(ERROR, "DISKQUEUE(%s) failed to sync - %s", d.name, err)
            }
            count = 0
        }

        // 有可读数据，并且当前读chan的数据已经被读走，则读取下一条数据
        if (d.readFileNum < d.writeFileNum) || (d.readPos < d.writePos) {
            if d.nextReadPos == d.readPos {
                dataRead, err = d.readOne()
                if err != nil {
                    d.logf(ERROR, "DISKQUEUE(%s) reading at %d of %s - %s",
                        d.name, d.readPos, d.fileName(d.readFileNum), err)
                    d.handleReadError()
                    continue
                }
            }
            r = d.readChan
        } else {
            // 如果无可读数据，那么设置 r 为nil, 防止将dataRead数据重复传入readChan中
            r = nil
        }

        select {
        // the Go channel spec dictates that nil channel operations (read or write)
        // in a select are skipped, we set r to d.readChan only when there is data to read
        case r <- dataRead:
            count++
            // moveForward sets needSync flag if a file is removed
            // 如果读chan 被写入成功，则会修改读的偏移
            d.moveForward()
        case <-d.emptyChan:
            // 清空所有文件，并返回empty的结果
            d.emptyResponseChan <- d.deleteAllFiles()
            count = 0
        case dataWrite := <-d.writeChan:
            // 写msg
            count++
            d.writeResponseChan <- d.writeOne(dataWrite)
        case <-syncTicker.C:
            // 到刷盘时间，则修改needSync = true
            if count == 0 {
                // avoid sync when there's no activity
                continue
            }
            d.needSync = true
        case <-d.exitChan:
            goto exit
        }
    }

exit:
    d.logf(INFO, "DISKQUEUE(%s): closing ... ioLoop", d.name)
    syncTicker.Stop()
    d.exitSyncChan <- 1
}
```

需要注意的点：

  1. 数据会预先读出来，当发送到readChan 里面，才会通过moveForward 操作更改读的偏移。
  2. queue 的Put 操作非操作，会等待写完成后，才会返回结果
  3. Empty 操作会清空所有数据
  4. 数据会定时或者按照设定的同步频次调用FSync 刷盘

### metadata 元数据

#### metadata 文件格式

文件名： `"name" + .diskqueue.meta.dat` 其中， name 是 topic, 或者topic + channel 命名.

metadata 数据包含5个字段, 内容如下：

```txt
    depth\nreadFileNum,readPos\nwriteFileNum,writePos
```

#### metadata 作用

当服务关闭后，metadata 数据将保存在文件中。当服务再次启动时，将从文件中将相关数据恢复到内存中。

## 学习总结

### 内存对齐与原子操作的问题

  ```go
  // 64bit atomic vars need to be first for proper alignment on 32bit platforms
  ```

- **现象** nsq 在定义struct 的时候，很多会出现类似的注释
- **原因** 原因在golang 源码 sync/atomic/doc.go 中

  ```go
  // On ARM, x86-32, and 32-bit MIPS,
  // it is the caller's responsibility to arrange for 64-bit
  // alignment of 64-bit words accessed atomically. The first word in a
  // variable or in an allocated struct, array, or slice can be relied upon to be
  // 64-bit aligned.
  ```

- **解释** 在arm, 32 x86系统，和 32位 MIPS 指令集中，调用者需要保证对64位变量做原子操作时是64位内存对齐的(而不是32位对齐)。而将64位的变量放在struct, array, slice 的最前面，可以保证64位对齐
- **结论** **有64bit 原子操作的变量，会定义在struct 的最前面，可以使变量使64位对齐，保证程序在32位系统中正确执行**

### 对象池的使用
   - `buffer_pool.go` 文件中, 简单实现了 bytes.Buffer 的对象池，减少了gc 压力
   - 使用场景，需要高频次做对象初始化和内存分配的情况，可使用sync.Pool 对象池减少gc 压力

### 如何将操作系统缓存中的数据主动刷新到硬盘中？
   - fsync 函数 (在write 函数之后，需要使用fsync 才能确保数据落盘)

> 本文代码来自于 [`github.com/nsqio/go-diskqueue`](http://github.com/nsqio/go-diskqueue)

