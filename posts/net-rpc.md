---
title: golang net/rpc 包的学习和使用
date: 2020-06-01 09:56:41
tags:
  - golang
  - net/rpc
---

从 rpc 包的 Server 端 和 Client 端入手，学习 Server/Client 的源码实现。并以一个例子作为总结。最后总结了rpc实现的几个学习的要点。

<!--more-->

# Golang net/rpc 包学习

golang 提供了一个开箱即用的RPC服务，实现方式简约而不简单。

## RPC 简单介绍

远程过程调用 (Remote Procedure Call，RPC) 是一种计算机通信协议。允许运行再一台计算机的程序调用另一个地址空间的子程序（一般是开放网络种的一台计算机），而程序员就像调用调用本地程序一样，无需额外的做交互编程。RPC 是一种 CS (Client-Server) 架构的模式，通过发送请求-接收响应的方式进行信息的交互。

有很多广泛使用的RPC框架，例如 gRPC, Thrift, Dubbo, brpc 等。这里的RPC 框架有的实现了跨语言调用，有的实现了服务注册发现等。比我们今天介绍的官方提供的 rpc 包要使用广泛的多。但是，通过对`net/rpc`的学习，可以使我们对一个rpc框架做一个最基本的了解。

## Golang 的实现

rpc 是 cs 架构，所以既有客户端，又有服务端。下面，我们先分析通信的编码，之后从服务端、客户端角度分析RPC的实现。

### 通信编码

golang 在rpc 实现中，抽象了协议层，我们可以自定义协议实现我们自己的接口。如下是协议的接口：

```go
// 服务端
type ServerCodec interface {
  ReadRequestHeader(*Request) error
  ReadRequestBody(interface{}) error
  WriteResponse(*Response, interface{}) error

  // Close can be called multiple times and must be idempotent.
  Close() error
}
// 客户端
type ClientCodec interface {
  WriteRequest(*Request, interface{}) error
  ReadResponseHeader(*Response) error
  ReadResponseBody(interface{}) error

  Close() error
}
```

而包中提供了基于gob 二进制编码的编解码实现。当然我们也可以实现自己想要的编解码方式。

### Server 端实现

#### 结构定义

```go
type Server struct {
  serviceMap sync.Map   // 保存Service
  reqLock    sync.Mutex // 读请求的锁
  freeReq    *Request
  respLock   sync.Mutex // 写响应的锁
  freeResp   *Response
}
```

server端通过互斥锁的方式支持了并发执行。由于每个请求和响应都需要定义Request/Response 对象，为了减少内存的分配，这里使用了一个freeReq/freeResp 链表实现了两个对象池。
当需要Request 对象时，从 freeReq 链表中获取，当使用完毕后，再放回链表中。

#### 服务的注册

service保存在 Server 的 serviceMap 中，每个Service 的信息如下：

```go
type service struct {
  name   string                 // 服务名
  rcvr   reflect.Value          // 服务对象
  typ    reflect.Type           // 服务类型
  method map[string]*methodType // 注册方法
}
```

从上面可以看到，一个类型以及该类型的多个方法可以被注册为一个Service。在注册服务时，通过下面的方法将服务保存在serviceMap 中。

```go
// 默认使用对象方法名
func (server *Server) Register(rcvr interface{}) error {}
// 指定方法名
func (server *Server) RegisterName(name string, rcvr interface{}) error {}
```

#### 服务的调用

首先，是rpc 服务的启动。和大部分的网络应用一致，在accept一个连接后，会启动一个协程做消息处理，代码如下：

```go
for {
  conn, err := lis.Accept()
  if err != nil {
    log.Print("rpc.Serve: accept:", err.Error())
    return
  }
  go server.ServeConn(conn)
}
```

其次，对于每一个连接，服务端会不断获取请求，并异步发送响应。代码如下：

```go
for {
  // 读取请求
  service, mtype, req, argv, replyv, keepReading, err := server.readRequest(codec)
  if err != nil {
    if debugLog && err != io.EOF {
      log.Println("rpc:", err)
    }
    if !keepReading {
      break
    }
    if req != nil {
      // 发送请求
      server.sendResponse(sending, req, invalidRequest, codec, err.Error())
      server.freeRequest(req)  // 释放 req 对象
    }
    continue
  }
  wg.Add(1)
  // 并发处理每个请求
  go service.call(server, sending, wg, mtype, req, argv, replyv, codec)
}
```

最后，由于异步发送请求，所以请求的顺序和响应顺序不一定一致。所以，在响应报文中，会携带请求报文的seq （序列号），保证消息的一致性。
除此之外，为了兼容http 服务，`net/rpc` 包还通过http包实现的 Hijack 方式，将 http 协议转换为 rpc 协议。代码如下：

```go
func (server *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
  // 客户端通过 CONNECT 方法连接

  // 通过Hijack 拿到tcp 连接
  conn, _, err := w.(http.Hijacker).Hijack()
  if err != nil {
    log.Print("rpc hijacking ", req.RemoteAddr, ": ", err.Error())
    return
  }
  // 发送客户端，支持 RPC 协议
  io.WriteString(conn, "HTTP/1.0 "+connected+"\n\n")

  // 开始 RPC 的请求响应
  server.ServeConn(conn)
}

```

### Client 端实现

客户端的连接相较于服务端是比较简单的。我们从发起连接、发送请求、读取响应三个角度学习。

#### RPC 的连接

由于该RPC支持HTTP协议做连接升级，因此，有几种连接方式。

1. 直接使用 tcp 协议。

    ```go
    func Dial(network, address string) (*Client, error) {}
    ```

2. 使用 http 协议。 http 协议可以指定路径，或者使用默认的rpc 路径。

    ```go
    // 默认路径 "/_goRPC_"
    func DialHTTP(network, address string) (*Client, error) {}
    // 使用默认的路径
    func DialHTTPPath(network, address, path string) (*Client, error) {}
    ```

#### 请求的发送

RPC 请求的发送，提供了同步和异步的接口调用，方式如下：

```go
// 异步
func (client *Client) Go(serviceMethod string, args interface{}, reply interface{}, done chan *Call) *Call {}
// 同步
func (client *Client) Call(serviceMethod string, args interface{}, reply interface{}) error{}
```

从内部实现可以知道，都是通过Go 异步的方式拿到返回数据。

下面，我们看内部如何实现请求的发送：

```go
func (client *Client) send(call *Call) {
  // 客户端正常的情况下
  seq := client.seq
  client.seq++  // 请求的序列号
  client.pending[seq] = call

    // 对请求进行编码，包括请求方法、参数。
  // Encode and send the request.
  client.request.Seq = seq
  client.request.ServiceMethod = call.ServiceMethod

  // client 可以并发 发起 Request, 然后异步等待 Done
    err := client.codec.WriteRequest(&client.request, call.Args)
    // 是否有发送失败，如果发送成功，则保存在pending map中，等待请求结果。
  if err != nil {
    client.mutex.Lock()
    call = client.pending[seq]
    delete(client.pending, seq)
    client.mutex.Unlock()
    if call != nil {
      call.Error = err
      call.done()
    }
  }
}
```

从上面可以看出，对于一个客户端，可以同时发送多条请求，然后异步等待响应。

#### 读取响应

在rpc 连接成功后，会建立一个连接，专门用于做响应的读取。

```go
for err == nil {
  response = Response{}
  err = client.codec.ReadResponseHeader(&response)
  if err != nil {
    break
  }
  seq := response.Seq
  client.mutex.Lock()
  call := client.pending[seq] // 从 pending 列表中删除
  delete(client.pending, seq)
  client.mutex.Unlock()
  // 解码body
  // 此处有多种判断，判断是否有异常
  client.codec.ReadResponseBody(nil)
  // 最后通知异步等待的请求，调用完成
  call.done()
}
```

通过循环读取响应头，响应body，并将读取结果通知调用rpc 的异步请求，完成一次响应的读取。

## 简单例子

下面我们官方提供的一个简单例子，对rpc包学习做个总结。

### 服务端

```go
type Args struct {  // 请求参数
  A, B int
}

type Quotient struct {  // 一个响应的类型
  Quo, Rem int
}

type Arith int

// 定义了乘法和除法
func (t *Arith) Multiply(args *Args, reply *int) error {
  *reply = args.A * args.B
  return nil
}

func (t *Arith) Divide(args *Args, quo *Quotient) error {
  if args.B == 0 {
    return errors.New("divide by zero")
  }
  quo.Quo = args.A / args.B
  quo.Rem = args.A % args.B
  return nil
}

func main() {
  serv := rpc.NewServer()
  arith := new(Arith)
  serv.Register(arith)  // 服务注册

    // 通过http 监听，到时做协议转换
  http.ListenAndServe("0.0.0.0:3000", serv)
}

```

### 客户端

```go
func main() {
  client, err := rpc.DialHTTP("tcp", "127.0.0.1:3000")
  if err != nil {
    log.Fatal("dialing:", err)
  }

  dones := make([]chan *rpc.Call, 0, 10)

  // 先同步发起请求
  for i := 0; i < 10; i++ {
    quotient := new(Quotient)
    args := &Args{i + 10, i}
    divCall := client.Go("Arith.Divide", args, quotient, nil)
    dones = append(dones, divCall.Done)
    log.Print("send", i)
  }
  log.Print("---------------")

  // 之后异步读取
  for idx, done := range dones {
    replyCall := <-done // will be equal to divCall
    args := replyCall.Args.(*Args)
    reply := replyCall.Reply.(*Quotient)
    log.Printf("%d / %d = %d, %d %% %d = %d\n", args.A, args.B, reply.Quo,
      args.A, args.B, reply.Rem)
    log.Print("recv", idx)
  }
}
```

## 我们可以学到什么

最后，做一个学习的总结。

1. 对统一连接上的不同请求实现异步操作，通过请求、响应需要保证数据的一致性。
2. 链表方式实现一个对象池
3. 对 http 包中实现的Hijack 方式的一次简单实践，通过http协议升级为rpc协议。劫持了原有http协议的tcp连接，转为rpc使用。
4. rpc 的实现，通过gob编码，应该是不支持与其他语言通信的。需要自己实现编解码方式。
5. rpc 的实现，也不支持服务的注册和发现，需要我们自己去维护服务方。

