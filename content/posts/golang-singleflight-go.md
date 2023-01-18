---
title: "Golang singleflight 使用和原理"
date: "2021-01-18"
tags:
  - golang
  - singleflight
---

本文介绍 `golang.org/x/sync/singleflight` 包的使用和原理。

<!--more-->

**建议结合[源码](https://github.com/golang/sync/tree/master/singleflight)阅读本文**

## 缓存击穿

在做高并发的服务时，不可避免的会遇到缓冲击穿的问题。缓冲击穿一般是说，当高并发流量缓存过期的情况下，出现大量请求从数据库读取相同数据的情况。这种情况下数据库的压力将瞬间增大。为了避免这种情况，一般有几种解决方案：
    1. 缓存永不过期，缓存做主动更新。
    2. 在使用缓存时，先检查缓存的过期时间，如果将要过期时，将过期时间延长到指定时间(避免其他服务也主动更新)，再主动做缓存更新(更新后，设置新的超时时间)。
    3. 加互斥锁，在db查询结束后，统一返回数据。(本文主要使用介绍用singleflight 来实现该方法) 这种方法的弊端是，只是单进程限制同时只能有一个请求。

## 如何使用 singleflight 解决缓存击穿

在singleflight 包中，提供了一个同时只运行一次方法(`fn`)的接口。这个接口和我们需要解决的缓存击穿问题异曲同工，下面简单介绍包中的几个方法：

```go
type Result struct {                                                               
    Val    interface{}                                                             
    Err    error                                                                   
    Shared bool                                                                    
}          

// 同步返回结果
func (g *Group) Do(key string, fn func() (interface{}, error)) (v interface{}, err error, shared bool) {}
// 返回channel，异步返回结果
func (g *Group) DoChan(key string, fn func() (interface{}, error)) <-chan Result {}
// 取新结果，不使用正在请求的结果
func (g *Group) Forget(key string){}
```

## 一个简单的例子

包中提供了同步访问和异步访问两种调用。我们需要用一个简单的例子来做说明:

```go
func main() {
  var count int32
  g := &singleflight.Group{}

  res := []<-chan singleflight.Result{}
  for i := 0; i < 10; i++ {
    key := "hello"
    res = append(res, g.DoChan("getdata", func() (interface{}, error) {
      // mock db query
      iter := atomic.AddInt32(&count, 1)
      time.Sleep(time.Duration(time.Microsecond))
      // return val
      return key + strconv.Itoa(int(iter)), nil
    }))
  }

  for i := 0; i < 10; i++ {
    dat := <-res[i]
    fmt.Println(dat.Val)
  }
}

```

上面例子中，我们 mock 了一个db读取的匿名方法，数据查询使用了1ms 的时间。返回值为 key + count 的值。
如果不用singleflight，我们取到的值，一定是key + 1...10，但是使用了之后，取到的结果都是 `hello1`在查询db时。仅查询了一次，将相同的查询归并为一条。结果可以证明singleflight 符合我们的预期，确实可以防止缓存击穿问题的发生。

## singleflight 的实现

通过对`singleflight`包的使用，推测`signleflight`的实现只需要在执行`fn`时，判断当前是否有正在进行的`fn`，如果存在则等待查询结果；如果没有，则记录并执行`fn`。**抽象的来说，就是希望同一个 `key` 指定的`fn`，同时仅执行一次，减少`fn`的调用次数**。
纸上得来终觉浅，绝知此事要躬行。下面看看这个包是怎么实现的：

### 错误类型的定义

首先是错误类型的定义， 除了正常的调用失败，一个方法的调用还可能包括 `panic` 错误 和 `runtime.Goexit` 调用，为了标记此类错误，因此定义了如下错误类型。

```go
var errGoexit = errors.New("runtime.Goexit was called")
type panicError struct {
    value interface{}
    stack []byte
}
```

### 执行中程序的调用

对于每一次执行fn，会构造一个call 结构体，用于将结果返回给等待的协程。doCall 则为 fn 的调用执行方法。

```go
type call struct {
    wg sync.WaitGroup
    val interface{}   // 调用的返回值
    err error      // 调用执行失败后的错误
    forgotten bool    // 标记是否下次调用时不使用正在调用的fn的结果
    dups  int        // 标记有多少调用方在等待fn 的结果
    chans []chan<- Result  // 等待结果的channal
}

func (g *Group) doCall(c *call, key string, fn func() (interface{}, error)) {
  normalReturn := false
  recovered := false

    // 为了能捕获到 goexit, 需要使用defer 来判断 （与panic 错误区分）
    // 实际上 goexit 无法捕获，只能通过标记 panic 和正常退出来排除
    // 第一个defer 是对执行结果的处理
  defer func() {
    // 非正常退出和panic， 则为 goexit 退出
    if !normalReturn && !recovered {
      c.err = errGoexit
    }

    c.wg.Done()
    g.mu.Lock()
    defer g.mu.Unlock()
    if !c.forgotten { 
      delete(g.m, key)
    }

    if e, ok := c.err.(*panicError); ok { // Panic 错误, 这种panic 无法捕获
      if len(c.chans) > 0 { 
        //对于 DoChan 的调用方式
        go panic(e)
        select {} // 保证 `go panic(e)` 的执行，并且 panic 无法被捕获。
      } else { 
        panic(e)
      }
    } else if c.err == errGoexit { // errGoexit 已经用排除法处理
    } else { 
      // 正常返回, 分发返回结果
      for _, ch := range c.chans {
        ch <- Result{c.val, c.err, c.dups > 0}
      }
    }
  }()

  // 执行 fn 方法，捕获 panic
  func() {
    defer func() {
      if !normalReturn {
        // 捕获 recover 错误
        if r := recover(); r != nil {
          c.err = newPanicError(r)
        }
      }
    }()

    c.val, c.err = fn()
    normalReturn = true
  }()

  if !normalReturn {
    // 如果非正常返回，则是通过 recover 的方式执行的 | 因为 goexit 方式不会走到这里
    recovered = true
  }
}
```

### 接口的实现

同步方式的调用：

```go
func (g *Group) Do(key string, fn func() (interface{}, error)) (v interface{}, err error, shared bool) {
  g.mu.Lock()
  if g.m == nil {
    g.m = make(map[string]*call)
  }
  if c, ok := g.m[key]; ok {
  // 执行中，则加入等待
    c.dups++
    g.mu.Unlock()
    c.wg.Wait()

    if e, ok := c.err.(*panicError); ok {
      panic(e)
    } else if c.err == errGoexit {
      runtime.Goexit()
    }
    return c.val, c.err, true
  }
  c := new(call)
  c.wg.Add(1)
  g.m[key] = c
  g.mu.Unlock()

  // 调用执行
  g.doCall(c, key, fn)
  return c.val, c.err, c.dups > 0
}
```

异步方式的调用：

```go
func (g *Group) DoChan(key string, fn func() (interface{}, error)) <-chan Result {
  ch := make(chan Result, 1)
  g.mu.Lock()
  if g.m == nil {
    g.m = make(map[string]*call)
  }
  if c, ok := g.m[key]; ok {
  // 如果正在执行，则加入到chan中，等待
    c.dups++
    c.chans = append(c.chans, ch)
    g.mu.Unlock()
    return ch
  }

  // 构造call
  c := &call{chans: []chan<- Result{ch}}
  c.wg.Add(1)
  g.m[key] = c
  g.mu.Unlock()

  // 异步执行
  go g.doCall(c, key, fn)

  return ch
}
```

设置下次调用不适用正在执行的结果

```go
func (g *Group) Forget(key string) {
  g.mu.Lock()
  if c, ok := g.m[key]; ok {
      c.forgotten = true
  }
  delete(g.m, key)
  g.mu.Unlock()
}
```

## 总结

从上述代码可以看出，做一个防止穿透的小功能，简单而不简约。需要考量的地方还是挺多(如何截获panic，如何判断goexit 等)。如下是内容总结：

1. `runtime.Goexit` 的特性, 以及如何捕获。(https://golang.org/cl/134395)
  > goexit 用于退出某个协程，但是之前注册的defer 方法仍然将会被执行。
2. 返回的error，不仅可能是正常逻辑错误，或者goexit 错误, 还有可能直接panic。
3. DoChan 调用方式，如果fn出现panic，该panic将无法被捕获，程序将退出。(Do 方式可以捕获panic), 而 Do 调用则可以捕获。例子如下：
```go
go func() {
  defer wg.Done()
  key := "hello"

  defer func() {
    if r := recover(); r != nil {
      fmt.Println("[[[", r, "]]]") // 此处可以捕获到panic. 由 doCall 方法中捕获后再次抛出的异常
    }
  }()
  _, _, _ = g.Do("getdata", func() (interface{}, error) {
    iter := atomic.AddInt32(&count, 1)
    time.Sleep(time.Duration(time.Microsecond))
    panic("panic")
    return key + strconv.Itoa(int(iter)), nil
  })
}()
```
