---
title: Golang Context
date: 2020-04-30 11:15:09
---

本文让我们一起来学习 golang Context 的使用和标准库中的Context的实现。

<!--more-->

golang context 包 一开始只是 Google 内部使用的一个 Golang 包，在 Golang 1.7的版本中正式被引入标准库。下面开始学习。

## 简单介绍

在学习 context 包之前，先看几种日常开发中经常会碰到的业务场景：

  1. 业务需要对访问的数据库，RPC ，或API接口，为了防止这些依赖导致我们的服务超时，需要针对性的做超时控制。
  2. 为了详细了解服务性能，记录详细的调用链Log。

上面两种场景在web中是比较常见的，context 包就是为了方便我们应对此类场景而使用的。

接下来, 我们首先学习 context 包有哪些方法供我们使用；接着举一些例子，使用 context 包应用在我们上述场景中去解决我们遇到的问题；最后从源码角度学习 context 内部实现，了解 context 的实现原理。

## Context 包

### Context 定义

context 包中实现了多种 Context 对象。Context 是一个接口，用来描述一个程序的上下文。接口中提供了四个抽象的方法，定义如下：

``` golang
type Context interface {
  Deadline() (deadline time.Time, ok bool)
  Done() <-chan struct{}
  Err() error
  Value(key interface{}) interface{}
}
```

- Deadline() 返回的是上下文的截至时间，如果没有设定，ok 为 false
- Done() 当执行的上下文被取消后，Done返回的chan就会被close。如果这个上下文不会被取消，返回nil
- Err() 有几种情况:
  - 如果Done() 返回 chan 没有关闭，返回nil
  - 如果Done() 返回的chan 关闭了， Err 返回一个非nil的值，解释为什么会Done()
    - 如果Canceled，返回 "Canceled"
    - 如果超过了 Deadline，返回 "DeadlineEsceeded"
- Value(key) 返回上下文中 key 对应的 value 值

### Context 构造

为了使用 Context，我们需要了解 Context 是怎么构造的。

Context 提供了两个方法做初始化：

``` golang
func Background() Context{}
func TODO() Context {}
```

上面方法均会返回空的 Context，但是 Background 一般是所有 Context 的基础，所有 Context 的源头都应该是它。TODO 方法一般用于当传入的方法不确定是哪种类型的 Context 时，为了避免 Context 的参数为nil而初始化的 Context。

其他的 Context 都是基于已经构造好的 Context 来实现的。一个 Context 可以派生多个子 context。基于 Context 派生新Context 的方法如下：

```go
func WithCancel(parent Context) (ctx Context, cancel CancelFunc){}
func WithDeadline(parent Context, d time.Time) (Context, CancelFunc) {}
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {}
```

上面三种方法比较类似，均会基于 parent Context 生成一个子 ctx，以及一个 Cancel 方法。如果调用了cancel 方法，ctx 以及基于 ctx 构造的子 context 都会被取消。不同点在于 WithCancel 必需要手动调用 cancel 方法，WithDeadline 可以设置一个时间点，WithTimeout 是设置调用的持续时间，到指定时间后，会调用 cancel 做取消操作。

除了上面的构造方式，还有一类是用来创建传递 traceId， token 等重要数据的 Context。

```go
func WithValue(parent Context, key, val interface{}) Context {}
```

withValue 会构造一个新的context，新的context 会包含一对 Key-Value 数据，可以通过Context.Value(Key) 获取存在 ctx 中的 Value 值。

通过上面的理解可以直到，Context 是一个树状结构，一个 Context 可以派生出多个不一样的Context。我们大概可以画一个如下的树状图：

![context_tree](context_tree.jpg)

一个background，衍生出一个带有traceId的valueCtx，然后valueCtx衍生出一个带有cancelCtx 的context。最终在一些db查询，http查询，rpc沙逊等异步调用中体现。如果出现超时，直接把这些异步调用取消，减少消耗的资源，我们也可以在调用时，通过Value 方法拿到traceId，并记录下对应请求的数据。

当然，除了上面的几种 Context 外，我们也可以基于上述的 Context 接口实现新的Context.

## 使用方法

下面我们举几个例子，学习上面讲到的方法。

### 超时查询的例子

在做数据库查询时，需要对数据的查询做超时控制，例如：

```go
ctx = context.WithTimeout(context.Background(), time.Second)
rows, err := pool.QueryContext(ctx, "select * from products where id = ?", 100)
```

上面的代码基于 Background 派生出一个带有超时取消功能的ctx，传入带有context查询的方法中，如果超过1s未返回结果，则取消本次的查询。使用起来非常方便。为了了解查询内部是如何做到超时取消的，我们看看DB内部是如何使用传入的ctx的。

在查询时，需要先从pool中获取一个db的链接，代码大概如下：

```go
// src/database/sql/sql.go
// func (db *DB) conn(ctx context.Context, strategy connReuseStrategy) *driverConn, error)

// 阻塞从req中获取链接，如果超时，直接返回
select {
case <-ctx.Done():
  // 获取链接超时了，直接返回错误
  // do something
  return nil, ctx.Err()
case ret, ok := <-req:
  // 拿到链接，校验并返回
  return ret.conn, ret.err
}
```

req 也是一个chan，是等待链接返回的chan，如果Done() 返回的chan 关闭后，则不再关心req的返回了，我们的查询就超时了。

在做SQL Prepare、SQL Query 等操作时，也会有类似方法：

```go
select {
default:
// 校验是否已经超时，如果超时直接返回
case <-ctx.Done():
  return nil, ctx.Err()
}
// 如果还没有超时，调用驱动做查询
return queryer.Query(query, dargs)
```

上面在做查询时，首先判断是否已经超时了，如果超时，则直接返回错误，否则才进行查询。

可以看出，在派生出的带有超时取消功能的 Context 时，内部方法在做异步操作（比如获取链接，查询等）时会先查看是否已经 Done了，如果Done，说明请求已超时，直接返回错误；否则继续等待，或者做下一步工作。这里也可以看出，要做到超时控制，需要不断判断 Done() 是否已关闭。

### 链路追踪的例子

在做链路追踪时，Context 也是非常重要的。（所谓链路追踪，是说可以追踪某一个请求所依赖的模块，比如db，redis，rpc下游，接口下游等服务，从这些依赖服务中找到请求中的时间消耗）

下面举一个链路追踪的例子：

```go
// 建议把key 类型不导出，防止被覆盖
type traceIdKey struct{}{}

// 定义固定的Key
var TraceIdKey = traceIdKey{}

func ServeHTTP(w http.ResponseWriter, req *http.Request){
  // 首先从请求中拿到traceId
  // 可以把traceId 放在header里，也可以放在body中
  // 还可以自己建立一个 （如果自己是请求源头的话）
  traceId := getTraceIdFromRequest(req)

  // Key 存入 ctx 中
  ctx := context.WithValue(req.Context(), TraceIdKey, traceId)

  // 设置接口1s 超时
  ctx = context.WithTimeout(ctx, time.Second)

  // query RPC 时可以携带 traceId
  repResp := RequestRPC(ctx, ...)

  // query DB 时可以携带 traceId
  dbResp := RequestDB(ctx, ...)

  // ...
}

func RequestRPC(ctx context.Context, ...) interface{} {
    // 获取traceid，在调用rpc时记录日志
    traceId, _ := ctx.Value(TraceIdKey)
    // request

    // do log
    return
}

```

上述代码中，当拿到请求后，我们通过req 获取traceId， 并记录在ctx中，在调用RPC，DB等时，传入我们构造的ctx，在后续代码中，我们可以通过ctx拿到我们存入的traceId，使用traceId 记录请求的日志，方便后续做问题定位。

当然，一般情况下，context 不会单纯的仅仅是用于 traceId 的记录，或者超时的控制。很有可能二者兼有之。

## 如何实现

知其然也需知其所以然。想要充分利用好 Context，我们还需要学习 Context 的实现。下面我们一起学习不同的 Context 是如何实现 Context 接口的，

### 空上下文

Background(), Empty() 均会返回一个空的 Context emptyCtx。emptyCtx 对象在方法 Deadline(), Done(), Err(), Value(interface{}) 中均会返回nil，String() 方法会返回对应的字符串。这个实现比较简单，我们这里暂时不讨论。

### 有取消功能的上下文

WithCancel 构造的context 是一个cancelCtx实例，代码如下。

```go
type cancelCtx struct {
  Context

  // 互斥锁，保证context协程安全
  mu       sync.Mutex
  // cancel 的时候，close 这个chan
  done     chan struct{}
  // 派生的context
  children map[canceler]struct{}
  err      error
}
```

WithCancel 方法首先会基于 parent 构建一个新的 Context，代码如下：

```go
func WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
  c := newCancelCtx(parent)  // 新的上下文
  propagateCancel(parent, &c) // 挂到parent 上
  return &c, func() { c.cancel(true, Canceled) }
}
```

其中，propagateCancel 方法会判断 parent 是否已经取消，如果取消，则直接调用方法取消；如果没有取消，会在parent的children 追加一个child。这里就可以看出，context 树状结构的实现。 下面是propateCancel 的实现：

```go
// 把child 挂在到parent 下
func propagateCancel(parent Context, child canceler) {
  // 如果parent 为空，则直接返回
  if parent.Done() == nil {
    return // parent is never canceled
  }
  
  // 获取parent类型
  if p, ok := parentCancelCtx(parent); ok {
    p.mu.Lock()
    if p.err != nil {
      // parent has already been canceled
      child.cancel(false, p.err)
    } else {
      if p.children == nil {
        p.children = make(map[canceler]struct{})
      }
      p.children[child] = struct{}{}
    }
    p.mu.Unlock()
  } else {
    // 启动goroutine，等待parent/child Done
    go func() {
      select {
      case <-parent.Done():
        child.cancel(false, parent.Err())
      case <-child.Done():
      }
    }()
  }
}
```

Done() 实现比较简单，就是返回一个chan，等待chan 关闭。可以看出 Done 操作是在调用时才会构造 chan done，done 变量是延时初始化的。

```go
func (c *cancelCtx) Done() <-chan struct{} {
  c.mu.Lock()
  if c.done == nil {
    c.done = make(chan struct{})
  }
  d := c.done
  c.mu.Unlock()
  return d
}

在手动取消 Context 时，会调用 cancelCtx 的 cancel 方法，代码如下：

func (c *cancelCtx) cancel(removeFromParent bool, err error) {
  // 一些判断,关闭 ctx.done chan
  // ...
  if c.done == nil {
    c.done = closedchan
  } else {
    close(c.done)
  }

  // 广播到所有的child，需要cancel goroutine 了
  for child := range c.children {
    // NOTE: acquiring the child's lock while holding parent's lock.
    child.cancel(false, err)
  }
  c.children = nil
  c.mu.Unlock()

  // 然后从父context 中，删除当前的context
  if removeFromParent {
    removeChild(c.Context, c)
  }
}
```

这里可以看到，当执行cancel时，除了会关闭当前的cancel外，还做了两件事，① 所有的child 都调用cancel方法，② 由于该上下文已经关闭，需要从父上下文中移除当前的上下文。

### 定时取消功能的上下文

WithDeadline, WithTimeout 提供了实现定时功能的 Context 方法，返回一个timerCtx结构体。WithDeadline 是给定了执行截至时间，WithTimeout 是倒计时时间，WithTImeout 是基于WithDeadline实现的，因此我们仅看其中的WithDeadline 即可。WithDeadline 内部实现是基于cancelCtx 的。相对于 cancelCtx 增加了一个计时器，并记录了 Deadline 时间点。下面是timerCtx 结构体：

```go
type timerCtx struct {
  cancelCtx
  // 计时器
  timer *time.Timer
  // 截止时间
  deadline time.Time
}
```

WithDeadline 的实现：

```go
func WithDeadline(parent Context, d time.Time) (Context, CancelFunc) {
  // 若父上下文结束时间早于child，
  // 则child直接挂载在parent上下文下即可
  if cur, ok := parent.Deadline(); ok && cur.Before(d) {
    return WithCancel(parent)
  }

  // 创建个timerCtx, 设置deadline
  c := &timerCtx{
    cancelCtx: newCancelCtx(parent),
    deadline:  d,
  }

  // 将context挂在parent 之下
  propagateCancel(parent, c)

  // 计算倒计时时间
  dur := time.Until(d)
  if dur <= 0 {
    c.cancel(true, DeadlineExceeded) // deadline has already passed
    return c, func() { c.cancel(false, Canceled) }
  }
  c.mu.Lock()
  defer c.mu.Unlock()
  if c.err == nil {
    // 设定一个计时器，到时调用cancel
    c.timer = time.AfterFunc(dur, func() {
      c.cancel(true, DeadlineExceeded)
    })
  }
  return c, func() { c.cancel(true, Canceled) }
}
```

构造方法中，将新的context 挂在到parent下，并创建了倒计时器定期触发cancel。

timerCtx 的cancel 操作，和cancelCtx 的cancel 操作是非常类似的。在cancelCtx 的基础上，做了关闭定时器的操作

```go
func (c *timerCtx) cancel(removeFromParent bool, err error) {
  // 调用cancelCtx 的cancel 方法 关闭chan，并通知子context。
  c.cancelCtx.cancel(false, err)
  // 从parent 中移除
  if removeFromParent {
    removeChild(c.cancelCtx.Context, c)
  }
  c.mu.Lock()
  // 关掉定时器
  if c.timer != nil {
    c.timer.Stop()
    c.timer = nil
  }
  c.mu.Unlock()
}
```

timeCtx 的 Done 操作直接复用了cancelCtx 的 Done 操作，直接关闭 chan done 成员。

## 传递值的上下文

WithValue 构造的上下文与上面几种有区别，其构造的context 原型如下：

```go
type valueCtx struct {
  // 保留了父节点的context
  Context
  key, val interface{}
}
```

每个context 包含了一个Key-Value组合。valueCtx 保留了父节点的Context，但没有像cancelCtx 一样保留子节点的Context. 下面是valueCtx的构造方法：

```go
func WithValue(parent Context, key, val interface{}) Context {
  if key == nil {
    panic("nil key")
  }
  // key 必须是课比较的，不然无法获取Value
  if !reflect.TypeOf(key).Comparable() {
    panic("key is not comparable")
  }
  return &valueCtx{parent, key, val}
}
```

直接将Key-Value赋值给struct 即可完成构造。下面是获取Value 的方法：

```go
func (c *valueCtx) Value(key interface{}) interface{} {
  if c.key == key {
    return c.val
  }
  // 从父context 中获取
  return c.Context.Value(key)
}
```

Value 的获取是采用链式获取的方法。如果当前 Context 中找不到，则从父Context中获取。如果我们希望一个context 多放几条数据时，可以保存一个map 数据到 context 中。这里不建议多次构造context来存放数据。毕竟取数据的成本也是比较高的。

## 注意事项

最后，在使用中应该注意如下几点：

- context.Background 用在请求进来的时候，所有其他context 来源于它。
- 在传入的conttext 不确定使用的是那种类型的时候，传入TODO context （不应该传入一个nil 的context)
- context.Value 不应该传入可选的参数，应该是每个请求都一定会自带的一些数据。（比如说traceId，授权token 之类的）。在Value 使用时，建议把Key 定义为全局const 变量，并且key 的类型不可导出，防止数据存在冲突。
- context goroutines 安全。

![](/images/weixin_logo.png)
