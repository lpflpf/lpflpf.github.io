---
title: Golang 限流器
date: 2020-06-20 14:17:00
tags:
  - golang
  - rate limiter
---

本文从源码角度学习不同限流器的实现方式。
<!--more-->

限流器是服务中非常重要的一个组件，在网关设计、微服务、以及普通的后台应用中都比较常见。它可以限制访问服务的频次和速率，防止服务过载，被刷爆。

限流器的算法比较多，常见的比如令牌桶算法、漏斗算法、信号量等。本文主要介绍基于漏斗算法的一个限流器的实现。文本也提供了其他几种开源的实现方法。

## 基于令牌桶的限流器实现

在golang 的官方扩展包 time 中（`github/go/time`），提供了一个基于令牌桶算法的限流器的实现。

### 原理

令牌桶限流器，有两个概念：

- 令牌：每次都需要拿到令牌后，才可以访问
- 桶：有一定大小的桶，桶中最多可以放一定数量的令牌
- 放入频率：按照一定的频率向通里面放入令牌，但是令牌数量不能超过桶的容量

因此，一个令牌桶的限流器，可以限制一个时间间隔内，最多可以承载桶容量的访问频次。下面我们看看官方的实现。

### 实现

#### 限流器的定义

下面是对一个限流器的定义：

```go
type Limiter struct {
  limit Limit // 放入桶的频率   （Limit 为 float64类型）
  burst int   // 桶的大小

  mu     sync.Mutex
  tokens float64 // 当前桶内剩余令牌个数
  last time.Time  // 最近取走token的时间
  lastEvent time.Time // 最近限流事件的时间
}
```

其中，核心参数是 limit，burst。 burst 代表了桶的大小，从实际意义上来讲，可以理解为服务可以承载的并发量大小；limit 代表了 放入桶的频率，可以理解为正常情况下，1s内我们的服务可以处理的请求个数。

在令牌发放后，会被保留在Reservation 对象中，定义如下：

```go
type Reservation struct {
  ok        bool  // 是否满足条件分配到了tokens
  lim       *Limiter // 发送令牌的限流器
  tokens    int   // tokens 的数量
  timeToAct time.Time  //  满足令牌发放的时间
  limit Limit  // 令牌发放速度
}
```

Reservation 对象，描述了一个在达到 timeToAct 时间后，可以获取到的令牌的数量tokens。 （因为有些需求会做预留的功能，所以timeToAct 并不一定就是当前的时间。

#### 限流器如何限流

官方提供的限流器有阻塞等待式的，也有直接判断方式的，还有提供了自己维护预留式的，但核心的实现都是下面的reserveN 方法。

```go
// 在 now 时间需要拿到n个令牌，最多可以等待的时间为maxFutureResrve
// 结果将返回一个预留令牌的对象
func (lim *Limiter) reserveN(now time.Time, n int, maxFutureReserve time.Duration) Reservation {
  lim.mu.Lock()

  // 首先判断是否放入频次是否为无穷大，如果为无穷大，说明暂时不限流
  if lim.limit == Inf {
    // ...
  }

  // 拿到截至now 时间时，可以获取的令牌tokens数量，上一次拿走令牌的时间last
  now, last, tokens := lim.advance(now)

  // 然后更新 tokens 的数量，把需要拿走的去掉
  tokens -= float64(n)

  // 如果tokens 为负数，说明需要等待，计算等待的时间
  var waitDuration time.Duration
  if tokens < 0 {
    waitDuration = lim.limit.durationFromTokens(-tokens)
  }

  // 计算是否满足分配条件
  // ① 需要分配的大小不超过桶容量
  // ② 等待时间不超过设定的等待时常
  ok := n <= lim.burst && waitDuration <= maxFutureReserve

  // 最后构造一个Reservation对象
  r := Reservation{
    ok:    ok,
    lim:   lim,
    limit: lim.limit,
  }
  if ok {
    r.tokens = n
    r.timeToAct = now.Add(waitDuration)
  }

  // 并更新当前limiter 的值
  if ok {
    lim.last = now
    lim.tokens = tokens
    lim.lastEvent = r.timeToAct
  } else {
    lim.last = last
  }

  lim.mu.Unlock()
  return r
}
```

从实现上看，limiter 并不是每隔一段时间更新当前桶中令牌的数量，而是记录了上次访问时间和当前桶中令牌的数量。当再次访问时，通过上次访问时间计算出当前桶中的令牌的数量，决定是否可以发放令牌。

### 使用

下面我们通过一个简单的例子，学习上面介绍的限流器的使用。

```go
  limiter := rate.NewLimiter(rate.Every(100*time.Millisecond), 10)
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    if limiter.Allow() {// do something
      log.Println("say hello")
    }
  })
  _ = http.ListenAndServe(":13100", nil)
```

上面，每100 ms 放入令牌桶中1个令牌，所以当批量访问该接口时，可以看到如下结果：

```shell
2020/06/26 14:34:16 say hello  有18 条记录
2020/06/26 14:34:17 say hello  有10 条记录
2020/06/26 14:34:18 say hello  有10 条记录
  ...
```

一开始漏斗满着，可以缓解部分突发的流量。当漏斗未空时，访问的频次和令牌放入的频次变为一致。

## 其他限流器的实现

1. uber 开源库中基于漏斗算法实现了一个[限流器](https://github.com/uber-go/ratelimit)。漏斗算法可以限制流量的请求速度，并起到削峰填谷的作用。
2. 滴滴开源实现了一个对http请求的[限流器中间件](https://github.com/didip/tollbooth)。可以基于以下模式限流。

   - 基于IP，路径，方法，header，授权用户等限流
   - 通过自定义方法限流
   - 还支持基于 http header 设置限流数据
   - 实现方式是基于 `github/go/time` 实现的，不同类别的数据都存储在一个带超时时间的数据池中。
3. golang 网络包中还有基于信号量实现的限流器,也值得我们去学习下。[源码地址](https://github.com/golang/net/blob/master/netutil/listen.go)。

## 总结

令牌桶实现的限流器算法，相较于漏斗算法可以在一定程度上允许突发的流量进入我们的应用中，所以在web应用中最为广泛。

在实际使用时，一般不会做全局的限流，而是针对某些特征去做精细化的限流。例如：通过header、x-forward-for 等限制爬虫的访问，通过对 ip,session 等用户信息限制单个用户的访问等。
