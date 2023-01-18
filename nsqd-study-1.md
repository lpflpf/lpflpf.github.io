---
title: 消息队列 NSQ 源码学习笔记 (一)
date: 2019-09-02 14:37:31
tags:
  - nsq
  - golang
  - messageQueue
  - nsqlookupd
---

> **nsqlookupd** 用于Topic, Channel, Node 三类信息的一致性分发


<!--more-->

# 概要

## nsqlookup 知识点总结
  - 功能定位 
    - 为node 节点和客户端节点提供一致的topic, channel, node 查询服务
      - **Topic** 主题， 和大部分消息队列的含义一致, 消息处理时，将相同主题的数据会归为一类消息
      - **channel**，可以理解为 topic 的一份数据拷贝，一个或者多个消费者对接一个channel。 
      - **node** nsqd 启动的一个实例
      - 一个channel会放置在某一个node 节点上，一个topic 下可以有多个channel. 
    - **HTTP 接口** 用于客户端服务发现以及admin 的交户使用
    - **TCP 接口**  用于 node 节点做消息广告使用
     
  - 实现方式
    - 数据包括了Topic, Channel, Node 等信息，全部存储于RegistrationDB中，RegistrationDB 采用读写锁和 map 实现，数据均存储于内存中
    - 若存在多个nsqlookup 节点，各节点之间无耦合关系
  
# nsqlookupd 源码阅读

程序入口文件: `/apps/nsqlookupd/main.go`

为了时NSQ 在windows 良好运行，NSQ 使用了 `github.com/judwhite/go-svc/svc` 包，用于构建一个可实现windows 服务。 可以用windows 的服务管理插件直接管理。

svc 包使用时，只需要实现 `github.com/judwhite/go-svc/svc.Service` 的接口即可。接口如下:

```go
type Service interface {
	// Init is called before the program/service is started and after it's
	// determined if the program is running as a Windows Service.
	Init(Environment) error

	// Start is called after Init. This method must be non-blocking.
	Start() error

	// Stop is called in response to os.Interrupt, os.Kill, or when a
	// Windows Service is stopped.
	Stop() error
}
```

因此，nsqlookup 只需要实现上述三个方法即可：

## Init 方法

此方法仅针对windows 的服务做了处理。若为windows 服务，则修改当前目录为可执行文件的目录。

## Stop 方法

此方法做了nsqlookupd.Exit() 的处理。
此处用到了sync.Once. 即调用的退出程序仅执行一次。

`Exit` 的具体内容为：

```go
func (l *NSQLookupd) Exit() {
	if l.tcpListener != nil {
		l.tcpListener.Close()
	}

	if l.httpListener != nil {
		l.httpListener.Close()
	}
	l.waitGroup.Wait()
}
```


1. 关闭 TCP Listener
2. 关闭 Http Listener
3. 等待所有goroutine的退出 (此处用到了sync.WaitGroup，用于等待goroutine 的退出)

## Start 方法

### 参数的初始化

  NSQ 命令行参数的构造，采用了golang 自带的flag 包。参数保存于Options对象中，采用了先初始化，后赋值的方式，减少了不必要的条件判断。  
  可以采用--config 的方式，直接添加配置文件。配置文件采用toml格式. 
  配置的解析，采用[`github.com/mreiferson/go-options`](//github.com/mreiferson/go-options) 实现，优先级由高到低为:

  - 命令行参数
  - deprecated 的命令行参数名称
  - 配置文件的值 (将命令行参数，连字符替换为下划线作为配置文件的key)
  - 若参数实现了Getter，则使用Get() 方法
  - 参数默认值

### 构造nsqlookupd
  
  - 初始化一个RegistrationDB
  - 建立 HttpListener 和 tcpListener (客户端请求)
  - 启动服务，等待连接请求或者中断信号

## RegistrationMap 的实现

```go
// RegistrationDB 使用读写锁做读写控制。
type RegistrationDB struct {
	sync.RWMutex
	registrationMap map[Registration]ProducerMap
}

type Registration struct {
	Category string   // Category 有三种类型，Topic, Channel, Client.
	Key      string
	SubKey   string
}

type ProducerMap map[string]*Producer

type Producer struct {
	peerInfo     *PeerInfo //客户端的相关信息
	tombstoned   bool
	tombstonedAt time.Time
}

type PeerInfo struct {
	lastUpdate       int64   // 上次更新的时间
	id               string  // 使用ip标识的id
	RemoteAddress    string `json:"remote_address"`
	Hostname         string `json:"hostname"`
	BroadcastAddress string `json:"broadcast_address"`
	TCPPort          int    `json:"tcp_port"`
	HTTPPort         int    `json:"http_port"`
	Version          string `json:"version"`
}
```



# 接口阅读

## TcpListener

> tcp 消息是 nsqd 与nsqlookupd 沟通的协议。 node 保存的是nsqd 的信息

Tcp Listener 是用来监听客户端发来的TCP 消息。  
建立连接后，发送4个byte标识连接的版本号。目前是v1. *"__V1"* (下划线用空格替代)
消息之间按照换行符`\n`分割。

目前客户端支持4类消息：
  - PING
    - 返回OK
    - 若存在对端的信息，则更新client.peerInfo.lastUpdate <上次更新时间>
  - IDENTIFY
    - 用于消息的认证，将nsqd信息发送给nsqlookupd.
    - 消息格式  `IDENTIFY\nBODYLEN(32bit)BODY`
        ```    
        |8bit    |1 bit | 32bit     | N bit |
        |IDENTIFY| 换行 | body 长度  | body  |
        ```
    - BODY 为json格式
    - 包含了如下字段：
        - 广播地址
        - TCP 端口
        - HTTP 端口
        - 版本号
        - 服务器地址 (通过连接直接获取)
  - REGISTER
    - 将nsqd 中注册的topic 和channel 信息发送到nsqlookupd 上，做信息共享
  - UNREGISTER
    - 将nsqd 中注销的topic 和channel 信息发送到nsqlookupd 上，做信息共享

## HTTPListener

> http 客户端的定位是用于服务的发现和admin的交互

1. 在学习 http 请求时，可以先学习下 `nsq/internal/http_api` 包，此包是对golang 中http请求handler 的一次封装：

```go

type Decorator func(APIHandler) APIHandler
type APIHandler func(http.ResponseWriter, *http.Request, httprouter.Params) (interface{}, error)

// f 是业务处理逻辑， ds 可以自定义多个包装器，用于对f 的输入和输出数据做处理。
func Decorate(f APIHandler, ds ...Decorator) httprouter.Handle {
	decorated := f
	for _, decorate := range ds {
		decorated = decorate(decorated)
	}
	return func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		decorated(w, req, ps)
	}
}

// Decorator 的一个例子，做日志记录的处理
func Log(logf lg.AppLogFunc) Decorator {
	return func(f APIHandler) APIHandler {
		return func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) (interface{}, error) {
			start := time.Now()
			response, err := f(w, req, ps)
			elapsed := time.Since(start)
			status := 200
			if e, ok := err.(Err); ok {
				status = e.Code
			}
			logf(lg.INFO, "%d %s %s (%s) %s",
				status, req.Method, req.URL.RequestURI(), req.RemoteAddr, elapsed)
			return response, err
		}
	}
}
```

这种处理方式类似于大部分web框架HTTP 中间件的处理方式，是利用递归嵌套的方式，保留了处理的上下文, 实现服务切片编程。

2. http 服务，使用[`github.com/julienschmidt/httprouter`](//github.com/julienschmidt/httprouter)包实现http 的路由功能。

3. 目前HTTP 客户端支持以下的请求:

| Method | Router                    | Param          |  Response |
|:------:|:-------------------------:|:---------------|:----------|
| GET    | /ping                     | -              |  "OK"     |
| GET    | /info                     | -              | 返回版本信息 |
| GET    | /debug                    | -              | 返回 db 中所有信息 |
| GET    | /lookup                   | topic          | 返回topic 关联的所有的channels 和 nsqd 服务的信息 |
| GET    | /topics                   | -              | 返回所有topic 的值 | 
| GET    | /channels                 | topic          | 返回topic 下所有的channels 信息 |
| GET    | /nodes                    | -              | 返回所有在线的nsqd 的node 信息, node 节点中包含了 topic 的信息，以及是否需要被删除|
| POST   | /topic/create             | topic          | 创建topic <不超过64个字符长度>|
| POST   | /topic/delete             | topic          | 删除相应topic 的channel 和topic 信息 |
| POST   | /channel/create           | topic, channel | 创建 channel ， 若topic 不存在，创建topic |
| POST   | /channel/delete           | topic, channel | 删除 channel， 支持 `*` |
| POST   | /topic/tombstone          | topic, node    | 将topic 下某个node 设置删除标识 *tombstone*, 给node 节点 一段空余时间用于删除相关topic 信息，并发送删除topic的命令| 
| GET    | /debug/pprof              | -              | pprof 提供的信息 | 
| GET    | /debug/pprof/cmdline      | -              | pprof 提供的信息 | 
| GET    | /debug/pprof/symbol       | -              | pprof 提供的信息 | 
| POST   | /debug/pprof/symbol       | -              | pprof 提供的信息 | 
| GET    | /debug/pprof/profile      | -              | pprof 提供的信息 | 
| GET    | /debug/pprof/heap         | -              | pprof 提供的信息 | 
| GET    | /debug/pprof/goroutine    | -              | pprof 提供的信息 | 
| GET    | /debug/pprof/block        | -              | pprof 提供的信息 | 
| GET    | /debug/pprof/threadcreate | -              | pprof 提供的信息 | 

# 学习总结
  - sync.Once， sync.RWMutex 读写锁的使用
  - http 包装函数的简单实现 nsq/internal/http_api.Decorate
  - [`github.com/judwhite/go-svc/svc`](//github.com/judwhite/go-svc) 的使用
  - [`github.com/julienschmidt/httprouter`](//github.com/julienschmidt/httprouter) 的使用
  - [`github.com/mreiferson/go-options`](//github.com/mreiferson/go-options) 的使用
