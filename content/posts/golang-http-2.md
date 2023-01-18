---
title: Golang Http 学习（二） Http Client 的实现
date: 2020-06-01 09:50:58
tags:
  - golang
---

golang http Client 的实现,  从 源码入手， 总结Client 的实现方式。

<!--more-->

> 众所周知，在golang 中实现的 http client 是自带连接池的。当我们做 http 请求时，极有可能就是复用了之前建立的 tcp 连接。那这个连接池是如何实现的，今天我们一起来探究。

## 请求操作

一个http 的请求操作，核心操作是通过构造一个 Request 对象，然后返回一个 Response 对象。
在 http 包中，http 的server 实现与client 的实现共用了Request/Response 对象。在 http client 中，我们通过构造Request，发起请求，并通过读取的数据构造Response 对象，返回给客户端的使用者；而在Server端，通过读取网络数据，通过数据头构造 Request 对象，并将响应数据放入 Response 对象中；通过将 Response 对象写入网络连接中，实现一次HTTP的交互。

在http client 的实现时，所有类型的http请求，均来自于如下方法：

```go
func (c *Client) do(req *Request) (retres *Response, reterr error) {}
```

do 方法中，考虑了重定向问题，以及请求cookie携带的相关问题。而最终发送 request 到获取 response ，来自于 RoundTriper 接口。该接口中仅有一个方法，是用来实现 Request 到 Response 转换的:

```go
type RoundTripper interface {
  RoundTrip(*Request) (*Response, error)
}
```

Transport 是我们最常用的 RoundTripper 接口的实现，它实现了http连接池的管理，连接的请求复用，并且是协程安全的。如果我们不指定，默认情况下 http 请求是使用 Transport 的实例 DefaultTransport 作为我们的 RoundTripper.

在 RoundTrip 中，抽象看来，主要有几个阶段：

1. 拿到一个连接
2. 发送Request，读取Response
3. 将连接返回给连接池

因此，下面我们从连接的管理维护、请求和响应的读写操作两个方面学习。

## 连接管理维护

连接的管理，Transport 中主要用到了如下的几个容器：

```go
// 保存连接池， 按照Key 区分连接池
idleConn     map[connectMethodKey][]*persistConn
// 等待连接的队列
idleConnWait map[connectMethodKey]wantConnQueue
// 空闲连接的LRU，用于删除最近未使用的连接
idleLRU      connLRU
// 保存每个host 目前的连接数
connsPerHost     map[connectMethodKey]int
// 当前等待 Dial 的连接数
connsPerHostWait map[connectMethodKey]wantConnQueue
```

从上述的几个容器可以看到，主要保存了当前正在使用的连接池，当前正在等待连接的队列，以及当前通过Dial 请求连接的池子等。这些容器使用的维度为connectMethodKey. 这个结构的定义如下：

```go
type connectMethodKey struct {
  // 代理，scheme，地址，
  proxy, scheme, addr string
  // 是否仅为 http1
  onlyH1              bool
}
```

可以看出，对于同一个connectMethodKey, 才会使用同一个连接池。
下面我们从获取一个连接开始，学习如何维护这个连接池。下面是获取连接的流程图：

![http-get-connection](http_get_conn.jpg)

1. 对于非Keeplive 的请求，则直接发起 Dial，不会复用连接。
2. 从上面的流程图中可以看到，我们的 wantConn 会放入两个队列 idleConnWait, connsPerHostWait。 当阻塞拿去连接时，如果有连接释放或者有新的连接成功连接，都会使我们拿到一个空闲连接。
3. 如果 Response 的 Body 关闭后，连接的读通道关闭，正常情况下会放入idleConn 连接池中。
4. 如果中间出现异常情况。例如：读操作失败，或者请求操作失败，该连接将不再被复用。
5. 如果在返回连接后，我们已经从idleConn 中拿到了一个连接，则返回后的连接将顺理成章的放入到空闲队列中。

在创建一个新的连接后，会启用两个新的goroutine： readLoop, writeLoop，用于连接的读操作和写操作。下面我们看看请求的读和写。

## 读写操作

http 请求，在同一个时刻是半双工的，要么是请求数据，要么是读取访问。在实现时，将读操作和写操作分别放在了不同的goroutine中，下面是一个从请求到Response 读取完成的时序图：

![golang-http-read_write](http_read_write.jpg)

从图中可以看出，整体操作分为如下几个步骤：

1. 传递一个 WriteRequest 对象至WriteLoop中，将请求通过连接发送到远端。
2. 同时会发送一个读 Response 的消息至readLoop，readLoop 开始阻塞读取远程数据
3. 读取成功数据后，readLoop 协程中将Response 返回至调用方。
4. 当关闭了Response 的Body 后，将通知readLoop。
5. 写成功后，会发送写成功的消息至readLoop, 告知该连接是正常的，可以继续复用。
6. 此时开始连接将复用，继续等待开始读的事件。（即 将连接收回至空闲连接池中，等待被重新触发请求）

## 总结

http 的连接池的实现就简单的介绍到这里。从上面连接池的实现，对我们使用时也有很多的启发：

1. 尽可能早的关闭 Response 的Body， 方便做连接的回收。
2. 连接池使用时，可以充分使用同一个Transport，使我们可以充分连接池。
3. 结合使用场景，在Transport 中设置空闲连接的超时时间，最大空闲连接数量，每个连接的最大连接数等值。

由于代码较多，这里不从代码角度分析。可以参考 “https://github.com/lpflpf/go” 中的注释。
