---
date: 2020-04-06 11:46:35
title: supervisor 的使用和进阶 (3)
subtitle: 路漫漫其修远兮，吾将上下而求索。
tags:
  - 服务管理
---

本文主要介绍 supervisor Event 的功能。

<!--more-->

> supervisor 作为一个进程管理工具，在 3.0 版本之后，新增了 Event 的高级特性, 主要用于做(进程启动、退出、失败等)事件告警服务。

Event 特性是将监听的服务(listener)注册到supervisord中，当supervisord监听到相应事件时，将事件信息推送给监听对应事件的listener。

### 事件类型

  Event 可以设置 27 种事件类型，可以分为如下几类：
    1. 监控进程状态转移事件;
    2. 监控进程状态日志变更事件;
    3. 进程组中进程添加删除事件;
    4. supervisord 进程本身日志变更事件;
    5. supervisord 进程本身状态变更的事件;
    6. 定时触发事件。

  事件可以被单独监听，也可以一个listener 监听多种事件。

### 配置说明

对于一个listener，与正常program的区别是，新增了events 参数，用于标识要监听的事件。

```
[eventlistener:theeventlistenername]
events=PROCESS_STATE,TICK_60 
buffer_size=10 ; 事件池子大小（输入流大小）
```

事件类型配置多个，用逗号分割。上述配置的是子进程状态的变更，以及定时60s通知间隔60s
事件通知缓冲区大小，可以自定义配置，上述配置了10个事件消息的缓冲。

### Listener 的实现

#### 与supervisord 的交互

由于supervisord 是 listener的父进程，所以交互方式采用最简单的 标准输入输出的方式交互。listener 通过标准输入获取事件，通过标准输出通知supervisord listener的事件处理结果，以及当前supervisord的状态

#### listener 的状态

listener 有三种状态：ACKNOWLEDGED、READY、BUSY.
  - ACKNOWLEDGED: listener 未就绪的状态。（发送READY之前的状态）
  - READY: 等待事件触发的状态。（发送READY 消息后，未收到消息的状态）
  - BUSY: 事件处理中的状态。（即输出 OK, FAIL 之前处理Event消息时的状态）

![](supervisor_listener_status.jpg)


#### 消息协议

消息包括supervisord 通知给listener 的事件消息和 listener 通知给supervisord 的状态变更消息。

listener 的状态变更消息, READY 
  - 状态OK的 "READY\n" 消息
  - 处理成功 "RESULT 2\nOK" 消息
  - 处理失败 "RESULT 4\nFAIL" 消息

supervisord 广播的事件消息, 事件消息分为 header 和 payload 两部分。 header 中采用kv的方式发送，header 中包含了 payload 的长度。

例如官网提供的header 的例子：

```
ver:3.0 server:supervisor serial:21 pool:listener poolserial:10 eventname:PROCESS_COMMUNICATION_STDOUT len:54
```

header 含义：

  - serial 为事件的序列号
  - pool 表示listener 的进程池名称(listener支持启动多个)
  - poolserial 表示listener的进程池序列号
  - eventname 事件名称
  - len body 的长度

#### Listener 的基本流程

listener 的处理流程如下：

    1. 发送ready消息，等待事件发生。
    2. 收到事件后，处理事件
    3. 事件处理完成后，发送 result 消息, 从第一步开始循环

### 进程状态转移举例
我们以进程状态转移作为例子，做简单介绍。

首先，使用 golang 实现listener

```go
package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

const RESP_OK = "RESULT 2\nOK"
const RESP_FAIL = "RESULT 4\nFAIL"

func main() {
	stdin := bufio.NewReader(os.Stdin)
	stdout := bufio.NewWriter(os.Stdout)
	stderr := bufio.NewWriter(os.Stderr)

	for {    
        // 发送后等待接收event
		_, _ = stdout.WriteString("READY\n")
		_ = stdout.Flush()
        // 接收header
		line, _, _ := stdin.ReadLine()          
		stderr.WriteString("read" + string(line))
		stderr.Flush()

		header, payloadSize := praseHeader(line)

		// 接收payload
		payload := make([]byte, payloadSize)
		stdin.Read(payload)   
		stderr.WriteString("read : " + string(payload))
		stderr.Flush()

		result := alarm(header, payload)

		if result {   // 发送处理结果
			stdout.WriteString(RESP_OK)
		} else {
			stdout.WriteString(RESP_FAIL)
		}
		stdout.Flush()
	}
}

func praseHeader(data []byte) (header map[string]string, 
        payloadSize int) {
	pairs := strings.Split(string(data), " ")
	header = make(map[string]string, len(pairs))

	for _, pair := range pairs {
		token := strings.Split(pair, ":")
		header[token[0]] = token[1]
	}

	payloadSize, _ = strconv.Atoi(header["len"])
	return header, payloadSize
}

// 这里设置报警即可
func alarm(header map[string]string, payload []byte) bool {
	// send mail
	return true
}
```
这里，报警处理未填写。

其次，在supervisor 中添加配置，监听服务:

```
[eventlistener:listener]
command=/root/listener
events=PROCESS_STATE,TICK_5
stdout_logfile=/var/log/tmp/listener_test_stdout.log
stderr_logfile=/var/log/tmp/listener_test_stderr.log
user=root
```

这里监听了服务的处理状态，以及每5s的心跳消息。

最后，启动listener。

```
supervisorct start listener
```

从stderr的日志中可以看到，简单的TICK_5 的消息(调整了格式):
```
header : ver:3.0 server:supervisor serial:256 pool:listener_test poolserial:173 eventname:TICK_5 len:15read 
payload: when:1586258030
```

fastcgi 进程状态变更的消息:

```
header : ver:3.0 server:supervisor serial:291 pool:listener_test poolserial:208 eventname:PROCESS_STATE_EXITED len:87
payload: processname:fastcgi_test groupname:fastcgi_test from_state:RUNNING expected:0 pid:19119

header :ver:3.0 server:supervisor serial:293 pool:listener_test poolserial:210 eventname:PROCESS_STATE_STARTING len:73
payload: processname:fastcgi_test groupname:fastcgi_test from_state:EXITED tries:0
```

[events 参考](http://supervisord.org/events.html)
![](/images/weixin_logo.png)
