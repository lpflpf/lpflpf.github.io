---
title: 消息队列 NSQ 源码学习笔记 (四)
date: 2019-09-10 18:09:55
tags:
  - nsq
  - golang
  - 消息队列
---

> **nsq** 工具集学习

<!--more-->

### nsq\_to\_nsq

nsq 作为消息队列，有个优势是nsqd 各节点之间是不关联的，如果一个节点出了问题，仅仅影响该节点下的topic，channel，以及相关的生产者、消费者。 也就是官方说明的特性第一条：**no SPOF** ( single point of failure 单点故障)。好处不言而喻，坏处也是有的，如果节点出问题，没有备份数据无法恢复。

所以，在官方提供了 nsq\_to\_nsq 作为 nsqd 节点复制的工具，用于做 nsqd 节点数据的备份, 或者也可以用于数据的分发。 类似于MirrorMaker.

#### 特性：
  - 支持将M 个 topic 的消息 publish 到 N 个 nsqd 上, 其中 M >= 1 , N >= 1. 也就是copy 是支持多对多的。
  - 多对多的模式支持两种：
    - **RoundRobin 模式**  对下游的nsqd 服务做轮询。
    - **HostPool 模式**  随机获取一个host，并发送

#### 总结

  由于nsqd 本身是不保序的，因此nsq\_to\_nsq 也是此特性，在复制数据和分发的时候，如果有多个接收的nsqd，并不能保证消息分发到相同的nsqd，因此无法保序。

### nsq\_to\_file

除了使用nsq\_to\_nsq做节点备份外，也可以通过数据落地的方式，做消息的物理备份。
nsq\_to\_file 可以将nsq接收到的数据，落地到硬盘。如需数据恢复，可以通过读取文件数据，重新生产即可。

### nsq\_tail

tail 查看 topic 的消息，打印topic 数据到标准输出。

### nsq\_stat

命令行打印服务的stats

```
---------------depth---------------+--------------metadata---------------
  total    mem    disk inflt   def |     req     t-o         msgs clients
  24660  24660       0     0    20 |  102688       0    132492418       1
  25001  25001       0     0    20 |  102688       0    132493086       1
  21132  21132       0     0    21 |  102688       0    132493729       1
```

### to\_nsq

命令行 push 消息到 topic , 默认换行符分割多条消息。
指定多个nsq，将同时向多个nsq 发布消息

### nsq\_to\_http

提供了一个 HTTP 推送的服务，将 TCP 消息的数据转化为 HTTP 请求，发送给消费端（支持 GET or POST 协议的web 服务）。
