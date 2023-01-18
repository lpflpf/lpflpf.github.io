---
title: Golang 熔断器
date: 2020-06-19 14:17:00
tags:
  - golang
  - circuit-breaker
---

本文主要从源码角度介绍golang 熔断器的一种实现。
<!--more-->

熔断器像是一个保险丝。当我们依赖的服务出现问题时，可以及时容错。一方面可以减少依赖服务对自身访问的依赖，防止出现雪崩效应；另一方面降低请求频率以方便上游尽快恢复服务。

熔断器的应用也非常广泛。除了在我们应用中，为了请求服务时使用熔断器外，在 web 网关、微服务中，也有非常广泛的应用。本文将从源码角度学习sony 开源的一个熔断器实现 `github/sony/gobreaker`。（代码注释可以从`github/lpflpf/gobreaker` 查看)

## 熔断器的模式

gobreaker是基于《微软云设计模式》一书中的熔断器模式的Golang实现。
下面是模式定义的一个状态机：

![state_machine](state_machine.png)

熔断器有三种状态，四种状态转移的情况：

**三种状态**：

- 熔断器关闭状态, 服务正常访问
- 熔断器开启状态，服务异常
- 熔断器半开状态，部分请求，验证是否可以访问

**四种状态转移**：

- 在熔断器关闭状态下，当失败后并满足一定条件后，将直接转移为熔断器开启状态。
- 在熔断器开启状态下，如果过了规定的时间，将进入半开启状态，验证目前服务是否可用。
- 在熔断器半开启状态下，如果出现失败，则再次进入关闭状态。
- 在熔断器半开启后，所有请求（有限额）都是成功的，则熔断器关闭。所有请求将正常访问。

## gobreaker 的实现

gobreaker 是在上述状态机的基础上，实现的一个熔断器。

### 熔断器的定义

```go
type CircuitBreaker struct {
  name          string
  maxRequests   uint32  // 最大请求数 （半开启状态会限流）
  interval      time.Duration   // 统计周期
  timeout       time.Duration   // 进入熔断后的超时时间
  readyToTrip   func(counts Counts) bool // 通过Counts 判断是否开启熔断。需要自定义
  onStateChange func(name string, from State, to State) // 状态修改时的钩子函数

  mutex      sync.Mutex // 互斥锁，下面数据的更新都需要加锁
  state      State  // 记录了当前的状态
  generation uint64 // 标记属于哪个周期
  counts     Counts // 计数器，统计了 成功、失败、连续成功、连续失败等，用于决策是否进入熔断
  expiry     time.Time // 进入下个周期的时间
}
```

其中，如下参数是我们可以自定义的：

- MaxRequests：最大请求数。当在最大请求数下，均请求正常的情况下，会关闭熔断器
- interval：一个正常的统计周期。如果为0，那每次都会将计数清零
- timeout: 进入熔断后，可以再次请求的时间
- readyToTrip：判断熔断生效的钩子函数
- onStateChagne：状态变更的钩子函数

### 请求的执行

熔断器的执行操作，主要包括三个阶段；①请求之前的判定；②服务的请求执行；③请求后的状态和计数的更新

```go

// 熔断器的调用
func (cb *CircuitBreaker) Execute(req func() (interface{}, error)) (interface{}, error) {

  // ①请求之前的判断
  generation, err := cb.beforeRequest()
  if err != nil {
    return nil, err
  }

  defer func() {
    e := recover()
    if e != nil {
      // ③ panic 的捕获
      cb.afterRequest(generation, false)
      panic(e)
    }
  }()

  // ② 请求和执行
  result, err := req()

  // ③ 更新计数
  cb.afterRequest(generation, err == nil)
  return result, err
}
```

### 请求之前的判定操作

请求之前，会判断当前熔断器的状态。如果熔断器以开启，则不会继续请求。如果熔断器半开，并且已达到最大请求阈值，也不会继续请求。

```go
func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
  cb.mutex.Lock()
  defer cb.mutex.Unlock()

  now := time.Now()
  state, generation := cb.currentState(now)

  if state == StateOpen { // 熔断器开启，直接返回
    return generation, ErrOpenState
  } else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests { // 如果是半打开的状态，并且请求次数过多了，则直接返回
    return generation, ErrTooManyRequests
  }

  cb.counts.onRequest()
  return generation, nil
}
```

其中当前状态的计算，是依据当前状态来的。如果当前状态为已开启，则判断是否已经超时，超时就可以**变更状态到半开**；如果当前状态为关闭状态，则通过周期判断是否进入下一个周期。

```go
func (cb *CircuitBreaker) currentState(now time.Time) (State, uint64) {
  switch cb.state {
  case StateClosed:
    if !cb.expiry.IsZero() && cb.expiry.Before(now) { // 是否需要进入下一个计数周期
      cb.toNewGeneration(now)
    }
  case StateOpen:
    if cb.expiry.Before(now) {
      // 熔断器由开启变更为半开
      cb.setState(StateHalfOpen, now)
    }
  }
  return cb.state, cb.generation
}
```

周期长度的设定，也是以据当前状态来的。如果当前正常（熔断器关闭），则设置为一个interval 的周期；如果当前熔断器是开启状态，则设置为超时时间（超时后，才能变更为半开状态）。

### 请求之后的处理操作

每次请求之后，会通过请求结果是否成功，对熔断器做计数。

```go
func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
  cb.mutex.Lock()
  defer cb.mutex.Unlock()

  now := time.Now()

  // 如果不在一个周期，就不再计数
  state, generation := cb.currentState(now)
  if generation != before {
    return
  }

  if success {
    cb.onSuccess(state, now)
  } else {
    cb.onFailure(state, now)
  }
}
```

如果在半开的状态下：

- 如果请求成功，则会判断当前连续成功的请求数 大于等于 maxRequests， 则可以把状态由**半开状态转移为关闭状态**
- 如果在半开状态下，请求失败，则会直接将**半开状态转移为开启状态**

如果在关闭状态下：

- 如果请求成功，则计数更新
- 如果请求失败，则调用readyToTrip 判断是否需要将状态**关闭状态转移为开启状态**

### 总结

- 对于频繁请求一些远程或者第三方的不可靠的服务，存在失败的概率还是非常大的。使用熔断器的好处就是可以是我们自身的服务不被这些不可靠的服务拖垮，造成雪崩。
- 由于熔断器里面，不仅会维护不少的统计数据，还有互斥锁做资源隔离，成本也会不少。
- 在半开状态下，可能出现请求过多的情况。这是由于半开状态下，连续请求成功的数量未达到最大请求值。所以，熔断器对于请求时间过长（但是比较频繁）的服务可能会造成大量的 `too many requests` 错误

----

1. 微软云设计模式(https://www.microsoft.com/en-us/download/details.aspx?id=42026)

