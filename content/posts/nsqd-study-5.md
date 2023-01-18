---
title: 消息队列 NSQ 源码学习笔记 (五)
date: 2019-09-12 11:08:31
tags:
  - nsq
  - golang
  - messageQueue
---

NSQ 的拓扑结构和生产消费端配置

<!--more-->
### 单机模式部署

NSQD 是可以脱离 nsqlookup 做单机部署的。
由于 nsqd 足够轻量，可以把服务部署在消息发布的服务器上，加快 pub 消息的速度，也能兼顾消费端消息的分发

### 集群模式

NSQD 是一个SPOF的系统，每个服务可以独立部署。当采用集群模式时，建议开启nsqlookup服务，用于管理多个 nsqd 的服务

一般的消息队列都会提供rebalance 的功能，nsqd 是没有的。 
不过可以通过nsq\_to\_nsq 做消息的复制，做服务的主备，当服务挂机后，可以切换到另外的服务器做消费。（中间channel 不会切换，因此可能会重复消费，或者丢一定消息）
nsqd 正常情况下，如果配置合理，消息是不会落地的。如果需要落地，可以使用nsq\_to\_file, 新建一个channel订阅 相关topic, 把消息落地到硬盘。

在集群模式下，可以部署多个 nsqlookupd 服务, 这些服务之间是互相没有依赖的，nsqd 在做消息广播的时候，会对每一个nsqlookupd的服务遍历一次，更新服务上的信息

### 生产

官方建议的生产方式，是通过 http 请求直接 pub 消息到nsq. 当然，大部分的nsq的客户端也实现了nsq 的消息发布功能

### 消费

消息队列的实现，一般都是推模型、拉模型或者推拉结合。在 nsq 中，是使用推模型，因此需要使用客户端来做消息的接收。
由于 nsq 消费的协议足够简单，也可以自行建立一个tcp 连接，做消息和连接的管理。

### 配置参数详解

#### nsqlookupd 配置

```
  -broadcast-address string            nsqd, client 访问的地址
    	address of this lookupd node, (default to the OS hostname) (default "tempt463.ops.shbt.qihoo.net")
  -http-address string                 http 监听
    	<addr>:<port> to listen on for HTTP clients (default "0.0.0.0:4161")
  -tcp-address string                  tcp 监听
    	<addr>:<port> to listen on for TCP clients (default "0.0.0.0:4160")

  -config string
    	path to config file

  -log-level value
    	set log verbosity: debug, info, warn, error, or fatal (default INFO)
  -log-prefix string
    	log message prefix (default "[nsqlookupd] ")

  -inactive-producer-timeout duration
    	duration of time a producer will remain in the active list since its last ping (default 5m0s)
  -tombstone-lifetime duration
    	duration of time a producer will remain tombstoned if registration remains (default 45s)

  -version
    	print version string
```

#### nsqd 配置

```
  -config string
    	path to config file
  -data-path string
    	path to store disk-backed messages

  -auth-http-address value             授权服务地址
    	<addr>:<port> to query auth server (may be given multiple times)

  -lookupd-tcp-address value    nslookupd tcp 地址, 广播使用
    	lookupd TCP address (may be given multiple times)
  -broadcast-address string     nslookupd 地址
    	address that will be registered with lookupd (defaults to the OS hostname) (default "tempt463.ops.shbt.qihoo.net")

  -tcp-address string
    	<addr>:<port> to listen on for TCP clients (default "0.0.0.0:4150")

  // http 服务
  -http-address string
    	<addr>:<port> to listen on for HTTP clients (default "0.0.0.0:4151")
  -http-client-connect-timeout duration
    	timeout for HTTP connect (default 2s)
  -http-client-request-timeout duration
    	timeout for HTTP request (default 5s)

  // https 服务
  -https-address string
    	<addr>:<port> to listen on for HTTPS clients (default "0.0.0.0:4152")
  -tls-cert string
    	path to certificate file
  -tls-client-auth-policy string
    	client certificate auth policy ('require' or 'require-verify')
  -tls-key string
    	path to key file
  -tls-min-version value
    	minimum SSL/TLS version acceptable ('ssl3.0', 'tls1.0', 'tls1.1', or 'tls1.2') (default 769)
  -tls-required
    	require TLS for client connections (true, false, tcp-https)
  -tls-root-ca-file string
    	path to certificate authority file

  // 日志
  -log-level value
    	set log verbosity: debug, info, warn, error, or fatal (default INFO)
  -log-prefix string
    	log message prefix (default "[nsqd] ")


  -msg-timeout duration
    	default duration to wait before auto-requeing a message (default 1m0s)
  -max-body-size int               命令消息体大小限制
    	maximum size of a single command body (default 5242880)
  -max-msg-size int                单条消息限制
    	maximum size of a single message in bytes (default 1048576)
  -max-msg-timeout duration        消息超时时间  touch 命令也不能超过该限制
    	maximum duration before a message will timeout (default 15m0s)
  -node-id int                     snowflake 算法中，生成message 的一部分, 为保证消息的唯一性，多个nsqd 需要不同的nodeid
    	unique part for message IDs, (int) in range [0,1024) (default is hash of hostname) (default 781)

  -max-rdy-count int               客户端可以批量处理的个数
    	maximum RDY count for a client (default 2500)
  -max-req-timeout duration        延时消息，最大可延时时间， 默认不超过1h
    	maximum requeuing timeout for a message (default 1h0m0s)
  -mem-queue-size int              内存队列大小
    	number of messages to keep in memory (per topic/channel) (default 10000)

  -max-channel-consumers int       每个channel 最多有多少个消费者
    	maximum channel consumer connection count per nsqd instance (default 0, i.e., unlimited)
  -max-heartbeat-interval duration 客户端的心跳, 默认间隔最大为1分钟
    	maximum client configurable duration of time between client heartbeats (default 1m0s)
  -max-output-buffer-size int      输出最大buffer
    	maximum client configurable size (in bytes) for a client output buffer (default 65536)
  -max-output-buffer-timeout duration 最大buffer 超时，如果时间超过，刷新到客户端
    	maximum client configurable duration of time between flushing to a client (default 30s)
  -min-output-buffer-timeout duration 最小buffer 超时时间, 尽量减少高频词写客户端
    	minimum client configurable duration of time between flushing to a client (default 25ms)
  -output-buffer-timeout duration  默认的客户端刷新时间， 可以通过 IDENTIFY 协议修改
    	default duration of time between flushing data to clients (default 250ms)

  stats 相关
  -statsd-address string
    	UDP <addr>:<port> of a statsd daemon for pushing stats
  -statsd-interval duration
    	duration between pushing to statsd (default 1m0s)
  -statsd-mem-stats
    	toggle sending memory and GC stats to statsd (default true)
  -statsd-prefix string
    	prefix used for keys sent to statsd (%s for host replacement) (default "nsq.%s")
  -statsd-udp-packet-size int
    	the size in bytes of statsd UDP packets (default 508)
  -e2e-processing-latency-percentile value
    	message processing time percentiles (as float (0, 1.0]) to track (can be specified multiple times or comma separated '1.0,0.99,0.95', default none)
  -e2e-processing-latency-window-time duration
    	calculate end to end latency quantiles for this duration of time (ie: 60s would only show quantile calculations from the past 60 seconds) (default 10m0s)

  diskqueue 相关
  -max-bytes-per-file int   // 磁盘队列单文件大小 默认 100M
    	number of bytes per diskqueue file before rolling (default 104857600)
  -sync-every int           // 默认不超过 2500 个消息将刷一次盘
    	number of messages per diskqueue fsync (default 2500)
  -sync-timeout duration    // 默认不超过 2s 将刷一次盘
    	duration of time per diskqueue fsync (default 2s)

  消息压缩
  -snappy
    	enable snappy feature negotiation (client compression) (default true)
  -deflate
    	enable deflate feature negotiation (client compression) (default true)
  -max-deflate-level int
    	max deflate compression level a client can negotiate (> values == > nsqd CPU usage) (default 6)

  -version
    	print version string
```

### 总结

总体来说，nsq 的优势在于足够轻量级，消费速度够快，没有单点问题。但缺点也显而易见：消息是不保序的，并且无法做自动的reblance.
