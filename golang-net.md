---
title: golang net 包学习
date: 2020-05-06 14:15:48
tags:
  - net
  - golang
---

golang 的 net 包，相关接口和结构比较多，今天做个简单的梳理。

<!--more-->

## 网络模型

在总结 net 包之前，还需要温习模糊的网络模型知识。下图是大学课本上的网络模型图：

![golang_net_model](golang_net_model.jpg)

模型图中可以看到，OSI 的七层模型，每一层实现的是与对端相应层的通信接口。但是实际应用中，我们把会话层、表示层、应用层统称为应用层。因此，就变成了TCP/IP 的五层模型。 其中网络层包含了 ip,arp,icmp 等协议，传输层包含了 TCP， UDP 等协议，应用层，比如 SMTP，DNS，HTTP 等协议。
在 net 包中，主要涉及网络层和传输层的协议。支持如下：
网络层：

- ICMP
- IGMP
- IVP6-ICMP

传输层：

- TCP
- UDP

## Socket 编程

在讲代码结构前，还需要回忆（学习）几个 Socket 编程(套接字编程)的知识点。

1. 在 Linux 上一切皆文件。所以各端口的读写服务可以认为是读取/写入文件, 一般使用文件描述符 fd (file descriptor) 表示。在Windows上，各端口的读写服务是一个通信链的句柄操作，通过句柄实现网络发出请求和读取数据。在 go 中为了统一，采用 linux 的 fd 代表一个链接节点。
2. TCP 是面向连接的、可靠的流协议，可以理解为不断从文件中读取数据（STREAM）。UDP 是无链接的、面向报文的协议，是无序，不可靠的（DGRAM）（目前很多可靠的协议都是基于UDP 开发的）。
3. UNIXDomain Socket 是一种 进程间通信的协议，之前仅在\*nix上使用，17年 17063 版本后支持了该协议。虽然是一个 IPC 协议，但是在实现上是基于套接字 (socket) 实现的。因此，UNIXDomain Socket 也放在了net 包中。
4. unixDomain Socket 也可以选择采用比特流的方式，或者无序的，不可靠的通讯方式，有序数据包的方式（SEQPACKET, Linux 2.6 内核才支持）

## 代码结构

下面我们看看 net 包中一些接口，以及一些接口的实现。

![golang_net_interface](golang_net_interface.jpg)

从图中可以看出，基于 TCP、UDP、IP、Unix （Stream 方式）的链接抽象出来都是 Conn 接口。基于包传递的 UDP、IP、UnixConn （DGRAM 包方式） 都实现了 PacketConn 接口。对于面向流的监听器，比如： TCPListener、 UnixListener 都实现了 Listener 接口。

整体上可以看出，net 包对网络链接是基于我们复习的网络知识实现的。对于代码的底层实现，也是比较简单的。正对不同的平台，调用不同平台套接字的系统调用即可。直观上看，对于不同的链接，我们都是可以通过Conn 的接口来做网络io的交互。

## 如何使用

在了解了包的构成后，我们基于不同的网络协议分两类来学习如何调用网络包提供的方法。

### 基于流的协议

基于流的协议，net 包中支持了常见的 TCP，Unix （Stream 方式） 两种。基于流的协议需要先于对端建立链接，然后再发送消息。下面是 Unix 套接字编程的一个流程：

![golang_net_stream](golang_net_stream.jpg)

首先，服务端需要绑定并监听端口，然后等待客户端与其建立链接，通过 Accept 接收到客户端的连接后，开始读写消息。最后，当服务端收到EOF标识后，关闭链接即可。 HTTP, SMTP 等应用层协议都是使用的 TCP 传输层协议。
### 基于包的协议

基于包的协议，net 包中支持了常见的 UDP，Unix （DGRAM 包方式，PacketConn 方式），Ip (网络层协议，支持了icmp, igmp) 几种。基于包的协议在bind 端口后，无需建立连接，是一种即发即收的模式。

基于包的协议，例如基于UDP 的 DNS解析， 文件传输（TFTP协议）等协议，在网络层应该都是基于包的协议。 下面是基于包请求的Server 端和Client端：

![golang_net_dgram](golang_net_dgram.jpg)

可以看到，在Socket 编程里， 基于包的协议是不需要 Listen 和 Accept 的。在 net 包中，使用ListenPacket，实际上仅是构造了一个UDP连接，做了端口绑定而已。端口绑定后，Server 端开始阻塞读取包数据，之后二者开始通信。由于基于包协议，因此，我们也可以采用PacketConn 接口（看第一个实现接口的图）构造UDP包。

## 一个简单的例子

下面，我们构造一个简单的 Redis Server （支持多线程），实现了支持Redis协议的简易Key-Value操作（可以使用Redis-cli直接验证）:

```go
package main

import (
  "bufio"
  "fmt"
  "io"
  "net"
  "strconv"
  "strings"
  "sync"
)

var KVMap sync.Map
func main() {
  // 构造一个listener
  listener, _ := net.Listen("tcp", "127.0.0.1:6379")
  defer func() { _ = listener.Close() }()
  for {
    // 接收请求
    conn, _ := listener.Accept()

    // 连接的处理
    go FakeRedis(conn)
  }
}

// 这里做了io 读写操作，并解析了 Redis 的协议
func FakeRedis(conn net.Conn) {
  defer conn.Close()
  reader := bufio.NewReader(conn)
  for {
    data, _, err := reader.ReadLine()
    if err == io.EOF {
      return
    }

    paramCount, _ := strconv.Atoi(string(data[1:]))
    var params []string
    for i := 0; i < paramCount; i++ {
      _, _, _ = reader.ReadLine() // 每个参数的长度，这里忽略了
      sParam, _, _ := reader.ReadLine()
      params = append(params, string(sParam))
    }

    switch strings.ToUpper(params[0]) {
    case "GET":
      if v, ok := KVMap.Load(params[1]); !ok {
        conn.Write([]byte("$-1\r\n"))
      } else {
        conn.Write([]byte(fmt.Sprintf("$%d\r\n%v\r\n", len(v.(string)), v)))
      }
    case "SET":
      KVMap.Store(params[1], params[2])
      conn.Write([]byte("+OK\r\n"))
    case "COMMAND":
      conn.Write([]byte("+OK\r\n"))
    }
  }

}
```

*上述代码没有任何的异常处理，仅作为网络连接的一个简单例子。*
从代码中可以看出，我们的数据流式的网络协议，在建立连接后，可以和文件IO服务一样，可以任意的读写操作。
正常情况下，流处理的请求，都会开启一个协程来做连接处理，主协程仅用来接收连接请求。(基于包的网络协议则可以不用开启协程处理)

## 总结

1. 基于 Conn 的消息都是有三种过期时间，这其实是在底层epoll\_wait中设置的超时时间。 Deadline 设置了Dail中建立连接的超时时间， ReadDeadline 是 Read 操作的超时时间， WriteDeadline 为 Write 操作的超时时间。
2. net 包作为基础包，基于net开发应用层协议比较多，例如 net/http,  net/rpc/smtp 等。
3. 网络的io操作底层是基于epoll来实现的, unixDomain 基于文件来实现的。
4. net 包实现的套接字编程仅是我们日常生活中用的比较多的一些方法，还有很多未实现的配置待我们去探索。
5. 网络模型比较简单，实际用起来，还是需要分门别类的。

