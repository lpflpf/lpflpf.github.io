---
title: 消息队列 NSQ 源码学习笔记 (三)
date: 2019-09-06 10:44:36
tags:
  - nsq
  - golang
  - 消息队列
---

> NSQD 源码学习

<!--more-->

# NSQD 学习笔记

## 特性总结

- 消息投放是不保序的 
  - 原因是内存队列、持久化队列、以及重新消费的数据混合在一起消费导致的
- 多个consumer 订阅同一个channel，消息将随机发送到不同的consumer 上
- 消息是可靠的
  - 当消息发送出去之后，会进入`in_flight_queue` 队列
  - 当恢复FIN 之后，才会从队列中将消费成功的消息清除
  - 如果客户端发送REQ，消息将会重发
- 消息发送采用的是推模式，减少延迟
- 支持延迟消费的模式: **DPUB**, 或者 **RRQ** (消费未成功，延时消费) 命令

## 代码学习

### 程序入口

程序入口 `github.com/nsq/apps/nsqd/main.go`

1. 获取配置，并从metadata 的持久化文件中读取topic、channel 信息。meta 信息格式:
```go
    Topics []struct {
        Name     string `json:"name"`
        Paused   bool   `json:"paused"`
        Channels []struct {
            Name   string `json:"name"`
            Paused bool   `json:"paused"`
        } `json:"channels"`
    } `json:"topics"`
```

2. 启动nsqd.Main 程序, 端口监听TCP 服务 和 HTTP 服务（支持HTTPS）。
3. 启动事件循环

- `queueScanLoop` 处理 `in-flight` 消息 和 `deferred` 消息队列事件的协程
- `lookupLoop` 处理与 nsqlookup 交互的协程。 包括消息的广播，lookup 节点的更新等。
- 如果配置了状态监听的地址，则会启动 `statsdLoop` 协程，用于定时发送(UDP)当前服务的各类状态

### Topic 处理

#### 数据结构

```go
type Topic struct {
    // 64bit atomic vars need to be first for proper alignment on 32bit platforms
    messageCount uint64    // 消息数量
    messageBytes uint64    // 消息字节数

    sync.RWMutex           // 结构体读写锁

    name              string
    channelMap        map[string]*Channel   // 保存topic 下所有channel
    backend           BackendQueue          // 落地的消息队列
    memoryMsgChan     chan *Message         // 内存中的消息
    startChan         chan int              // topic 被订阅了，可以启动消费了
    exitChan          chan int              // 协程退出channel
    channelUpdateChan chan int              // channel 更新的消息
    waitGroup         util.WaitGroupWrapper 
    exitFlag          int32                 // 退出标记
    idFactory         *guidFactory          // uuid 生成器

    ephemeral      bool                     // 是否为临时topic
    deleteCallback func(*Topic)             // 临时topic，自动删除相关channel
    deleter        sync.Once

    paused    int32
    pauseChan chan int                      // 暂停的信号

    ctx *context                            // 上下文，保存nsqd 
}
```

#### topic 的创建

- 初始化内存队列
- 初始化diskqueue
- 初始化topic 相应的 msg 唯一id生成器
- 向nsqlookup 广播，添加topic信息
- 等待事件处理 (consumer 和 channel 相关)
  - 只有consumer 存在topic 订阅(Sub)之后，才会启动 Topic 的事件处理

    ```go
        for {
            select {
            // 消息可从二者中随机获取，所以topic 中消息是不保序的
            case msg = <-memoryMsgChan:   // 内存消息
            case buf = <-backendChan:     // 持久化文件中推送的消息
                msg, err = decodeMessage(buf)
                if err != nil {
                    t.ctx.nsqd.logf(LOG_ERROR, "failed to decode message - %s", err)
                    continue
                }
            case <-t.channelUpdateChan:   // 更新channel，则会增加topic 下发的列表
                chans = chans[:0]
                t.RLock()
                for _, c := range t.channelMap {
                    chans = append(chans, c)
                }
                t.RUnlock()
                if len(chans) == 0 || t.IsPaused() {
                    memoryMsgChan = nil
                    backendChan = nil
                } else {
                    memoryMsgChan = t.memoryMsgChan
                    backendChan = t.backend.ReadChan()
                }
                continue
            case <-t.pauseChan:           // 暂停topic，则所有chan 都暂停
                if len(chans) == 0 || t.IsPaused() {
                    memoryMsgChan = nil
                    backendChan = nil
                } else {
                    memoryMsgChan = t.memoryMsgChan
                    backendChan = t.backend.ReadChan()
                }
                continue
            case <-t.exitChan:
                goto exit
            }
    
            for i, channel := range chans {   // 将topic 收到的消息广播到 topic 下所有的channel 中
                chanMsg := msg

                // 考虑比较周全的是，减少一次message 的创建
                if i > 0 {                    
                    chanMsg = NewMessage(msg.ID, msg.Body)
                    chanMsg.Timestamp = msg.Timestamp
                    chanMsg.deferred = msg.deferred
                }
                if chanMsg.deferred != 0 {    // 如果是defer 的消息，会添加到channel 的defer 队列中
                    channel.PutMessageDeferred(chanMsg, chanMsg.deferred)
                    continue
                }
                err := channel.PutMessage(chanMsg)  // 正常消息，直接添加到channel 中
                if err != nil {
                    t.ctx.nsqd.logf(LOG_ERROR,
                        "TOPIC(%s) ERROR: failed to put msg(%s) to channel(%s) - %s",
                        t.name, msg.ID, channel.name, err)
                }
            }
        }
    ```

#### 值得关注的topic 操作

- putMessage

  ```go
func (t *Topic) put(m *Message) error {
    select {
    case t.memoryMsgChan <- m:
    default:
        b := bufferPoolGet()
        err := writeMessageToBackend(b, m, t.backend)
        bufferPoolPut(b)
        t.ctx.nsqd.SetHealth(err)
        if err != nil {
            t.ctx.nsqd.logf(LOG_ERROR,
                "TOPIC(%s) ERROR: failed to write message to backend - %s",
                t.name, err)
            return err
        }
    }
    return nil
}
  ```

  利用了golang chan 阻塞的原理，当 memoryMsgChan 满了之后，`case t.memoryMsgChan <- m` 无法执行，会执行 `default` 操作，自动添加消息到硬盘中。

- messageId 的生成，使用了业界常用的snowflake 算法。

### Channel 处理

channel 没有自己的事件操作，都是通过被动执行相关操作。

#### 数据结构

```go
type Channel struct {
    // 64bit atomic vars need to be first for proper alignment on 32bit platforms
    requeueCount uint64             // 重新消费的message 个数
    messageCount uint64             // 消息总数
    timeoutCount uint64             // 消费超时的message 个数

    sync.RWMutex

    topicName string
    name      string
    ctx       *context

    backend BackendQueue            // 落地的队列

    memoryMsgChan chan *Message     // 内存中的消息
    exitFlag      int32             // 退出标识
    exitMutex     sync.RWMutex

    // state tracking
    clients        map[int64]Consumer // 支持多个client 消费，但是一条消息仅能被某一个client 消费
    paused         int32            // 暂停标识
    ephemeral      bool             // 临时 channel 标识
    deleteCallback func(*Channel)   // 删除的回调函数
    deleter        sync.Once

    // Stats tracking
    e2eProcessingLatencyStream *quantile.Quantile

    // TODO: these can be DRYd up
    deferredMessages map[MessageID]*pqueue.Item    // defer 消息保存的map
    deferredPQ       pqueue.PriorityQueue          // defer 队列 (优先队列保存)
    deferredMutex    sync.Mutex                    // 相关的互斥锁

    inFlightMessages map[MessageID]*Message        // 正在消费的消息保存的map
    inFlightPQ       inFlightPqueue                // 正在消费的消息保存在优先队列 (优先队列保存)
    inFlightMutex    sync.Mutex                    // 相关的互斥锁
}
```

### 事件循环处理

在启动nsqd 时，会启动一些事件循环的处理。

#### channel 队列处理
   
  channel 有有两个重要队列： defer队列和inflight 队列, 事件处理主要是对两个队列的消息数据做处理

  - **扫描channel 规则**
    - 更新 channels 的频率为100ms
    - 刷新表的频率为 5s
    - 默认随机选择20( **queue-scan-selection-count** ) 个channels 做消息队列调整
    - 默认处理队列的协程数量不超过 4 ( **queue-scan-worker-pool-max** )
  - **processInFlightQueue**  做消息处理超时重发处理
    - flight 队列中，保存的是推送到消费端的消息，优先队列中，按照time排序, 消息已经发送的时间越久越靠前
    - 定时从flight 队列中获取最久的消息，如果已超时( 超过 **msg--time** )，则将消息**重新发送**
  - **processDeferdQueue** 处理延迟队列的消息
    - deferd 队列中，保存的是延迟推送的消息，优先队列中，按照time排序，距离消息要发送的时间越短，越靠前
    - 定时从deferd 队列中获取最近需要发送的消息，如果消息已达到发送时间，则pop 消息，将消息**发送**

#### lookup 事件响应

  此处的事件循环，是用于和lookupd 交户使用的事件处理模块。例如Topic 增加或者删除， channel 增加或者删除 需要对所有 nslookupd 模块做消息广播等处理逻辑，均在此处实现。
  主要的事件:

  - **定时心跳操作** 每隔 15s 发送 PING 到 所有 nslookupd 的节点上
  - **topic,channel新增删除操作** 发送消息到所有 nslookupd 的节点上
  - **配置修改的操作** 如果配置修改，会重新从配置中刷新一次 nslookupd 节点

### 消费协程事件处理

  当一个客户端与nsqd 通过TCP建立连接后，将启动protocolV2.messagePump 协程，用于处理消息的交户,主协程用于做事件的响应。

  messagePump:
    
  ```go
func (p *protocolV2) messagePump(client *clientV2, startedChan chan bool) {
    var err error
    var memoryMsgChan chan *Message
    var backendMsgChan chan []byte
    var subChannel *Channel
    var flusherChan <-chan time.Time
    var sampleRate int32

    subEventChan := client.SubEventChan
    identifyEventChan := client.IdentifyEventChan
    outputBufferTicker := time.NewTicker(client.OutputBufferTimeout)
    heartbeatTicker := time.NewTicker(client.HeartbeatInterval) // 客户端超时时间的一半， 默认为30s
    heartbeatChan := heartbeatTicker.C
    msgTimeout := client.MsgTimeout

    flushed := true

    close(startedChan)

    for {
        if subChannel == nil || !client.IsReadyForMessages() {
            memoryMsgChan = nil
            backendMsgChan = nil
            flusherChan = nil
            client.writeLock.Lock()
            err = client.Flush()
            client.writeLock.Unlock()
            if err != nil {
                goto exit
            }
            flushed = true
        } else if flushed {
            memoryMsgChan = subChannel.memoryMsgChan
            backendMsgChan = subChannel.backend.ReadChan()
            flusherChan = nil                    
        } else {
            memoryMsgChan = subChannel.memoryMsgChan
            backendMsgChan = subChannel.backend.ReadChan()
            flusherChan = outputBufferTicker.C   // 如果动态设置了flusher 的定时器，则使用这个定时器刷新
        }

        select {
        case <-flusherChan:
            client.writeLock.Lock()
            err = client.Flush()   // 把writer flush
            client.writeLock.Unlock()
            if err != nil {
                goto exit
            }
            flushed = true
        case <-client.ReadyStateChan:
        case subChannel = <-subEventChan:  // 一个consumer 同一个tcp 连接，只能订阅一个topic
            subEventChan = nil
        case identifyData := <-identifyEventChan:  // 客户端认证
            identifyEventChan = nil

            outputBufferTicker.Stop()
            if identifyData.OutputBufferTimeout > 0 {
                outputBufferTicker = time.NewTicker(identifyData.OutputBufferTimeout)
            }

            heartbeatTicker.Stop()
            heartbeatChan = nil
            if identifyData.HeartbeatInterval > 0 {   // 设置刷新时间
                heartbeatTicker = time.NewTicker(identifyData.HeartbeatInterval)
                heartbeatChan = heartbeatTicker.C
            }

            if identifyData.SampleRate > 0 {  // 可以设置采样数据，采样输出数据
                sampleRate = identifyData.SampleRate
            }

            msgTimeout = identifyData.MsgTimeout   // identify 可以设置消息的超时事件
        case <-heartbeatChan:       // 心跳消息
            err = p.Send(client, frameTypeResponse, heartbeatBytes)
            if err != nil {
                goto exit
            }
        case b := <-backendMsgChan:  // 硬盘消息推送到consumer 中
            if sampleRate > 0 && rand.Int31n(100) > sampleRate {
                continue
            }

            msg, err := decodeMessage(b)  // 硬盘消息保存为二进制，需要解码
            if err != nil {
                p.ctx.nsqd.logf(LOG_ERROR, "failed to decode message - %s", err)
                continue
            }
            msg.Attempts++

            // 设置超时事件，并将消息放入flight 队列中
            subChannel.StartInFlightTimeout(msg, client.ID, msgTimeout)
            client.SendingMessage()
            err = p.SendMessage(client, msg)
            if err != nil {
                goto exit
            }
            flushed = false
        case msg := <-memoryMsgChan:   // 内存消息推送到consumer
            if sampleRate > 0 && rand.Int31n(100) > sampleRate {
                continue
            }
            msg.Attempts++

            // 设置超时事件，并将消息放入flight 队列中
            subChannel.StartInFlightTimeout(msg, client.ID, msgTimeout)
            client.SendingMessage()
            err = p.SendMessage(client, msg)
            if err != nil {
                goto exit
            }
            flushed = false
        case <-client.ExitChan:
            goto exit
        }
    }

exit:
    p.ctx.nsqd.logf(LOG_INFO, "PROTOCOL(V2): [%s] exiting messagePump", client)
    heartbeatTicker.Stop()
    outputBufferTicker.Stop()
    if err != nil {
        p.ctx.nsqd.logf(LOG_ERROR, "PROTOCOL(V2): [%s] messagePump error - %s", client, err)
    }
}
  ```

### HTTP 传输协议

| METHOD | ROUTE  | PARAM | INFO |
|:----:|:----|:-----|:-----|
|  GET   | /ping  |   -   | 如果服务器正常，返回 OK |
|  GET   | /info  |   -   | 返回服务器的相关信息    |
|  POST  | /pub   | topicName, [defer] | 消息发布, 可以选择 defer 发布 |
|  POST  | /mpub  | topicName, [binary] | 多条消息的发布， 可以支持二进制消息的发布, 消息格式为 (msgNum + (msgSize + msg) * msgNum), 非binary 模式，则按照换行符分割消息 |
|  GET   | /stats | format, topic, channel, include_clients | 获取响应服务的状态， 可以通过topic, channel 过滤. |
|  POST  | /topic/create | topic | 创建一个 topic |
|  POST  | /topic/delete | topic | 删除一个 topic |
|  POST  | /topic/empty  | topic | 清空一个 topic |
|  POST  | /topic/pause  | topic | 暂停一个 topic |
|  POST  | /topic/unpause | topic | 启动一个暂停的 topic |
|  POST  | /channel/create | topic, channel | 创建一个 channel |
|  POST  | /channel/delete | topic, channel | 删除一个 channel |
|  POST  | /channel/empty  | topic, channel | 清空一个channel， 包括 内存中的队列 和 硬盘中的队列 |
|  POST  | /channel/pause  | topic, channel | 暂停一个 channel |
|  POST  | /channel/unpause| topic, channel | 启动一个暂停的 channel |
|  PUT   | /config/:opt    | nsqlookupd_tcp_addresses, log_level | 修改 nsqlookupd 的地址，或者 日志级别 |
|  GET/POST | /debug | - | something |

### TCP 传输协议

| PROTOCAL  |  PARAM           | 解释 |
|:------:|:----|:----|
| IDENTIFY  |  Body (len + data)               | 客户端认证, body 采用 json 格式, 主要提供消息消费相关参数信息 |
| FIN       | msgId            | 消息消费完成 | 
| RDY       | size             | 若客户端准备好接收消息，将发送RDY 命令，设置消费端可等待的消息量（类似批量消息）。设置为0，则暂停接收 |
| REQ       | msgId, timeoutMs | 将 in_flight_queue 队列中的消息放到 deferd 队列中，延时消费 （可以认为是消费失败的消息的一种处理方式） |
| PUB       | topicName , Body (len + msg) | 消息生产者发布消息到Topic 队列   | 
| MPUB      | TopicName, Body （len + msgNum + (msgSize + msg ) * msgNum | 消息生产着发布多条消息到Topic队列  |
| DPUB      | TopicNmae, timeoutMs, Body (len + msg) | 消息生产者发布定时消息到Topic 队列  |
| NOP       | -                | 空消息  |
| TOUCH     | msgId            | 重置在 in_flight_queue 队列中的消息的超时时间 |
| SUB       | topicName, channelName | 消费端通过某个channel订阅某个topic 消息，订阅成功后，将通过 messagePump 推送消息到消费端 |
| CLS       | -                | 消费端暂停接收消息, 等待关闭 |
| AUTH      | body             | 授权  |

## 学习总结
  - nsqd 消息id 生成方法采用的uuid 生成算法 snowflake 算法
  - `in_flight_queue` 和 `delay_queue` 实现都是使用堆排序实现的优先队列
  - 从M 个channel 中随机筛选N个channel 做队列队列扫描, 每次获取的概率相同

   ```go
func UniqRands(quantity int, maxval int) []int {
    if maxval < quantity {
        quantity = maxval
    }

    intSlice := make([]int, maxval)
    for i := 0; i < maxval; i++ {
        intSlice[i] = i
    }

    // 每次从[i, maxval] 中筛选 1 个元素，放到位置 i 中
    for i := 0; i < quantity; i++ {
        j := rand.Int()%maxval + i
        // swap
        intSlice[i], intSlice[j] = intSlice[j], intSlice[i]
        maxval--

    }
    return intSlice[0:quantity]
}
   ```
