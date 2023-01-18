---
date: 2020-04-06 10:46:35
title: supervisor 的使用和进阶 (2)
subtitle: 路漫漫其修远兮，吾将上下而求索。
tags:
  - 服务管理
---

本文主要介绍 supervisor 对 fastcgi 进程的管理

<!--more-->

## fastcgi 进程的管理

在php 中，php-fpm 有主进程来管理和维护子进程的数量。但是并不是所有的服务都有类似的主进程来做子进程的维护。
在很多其他语言中，有很多比较有名的fastcgi 服务，例如py 的flup， c++ 实现的 FastCgi++等。如果这些服务在单机中启动多个进程（极有可能），那如何管理这些进程是个比较头疼的问题。  supervisor 的fastcgi 管理的功能就是为了解决这个问题。

### 配置

在普通进程的基础上，添加如下配置：

```
[fcgi-program:x]

socket = "tcp://10.3.2.10:9002"     // 支持 tcp ，或者 Unix socket
socket_backlog = 1024               // 2 的N次方, 根据机器配置设置, 默认是端口最大监听量
socket_owner = chrism:wheel         // 监听用户组
socket_mode = 0700                  // 监听模式
```

### 举个例子

#### 实现一个简单的fastcgi 服务

通过监听127.0.0.1:9001 端口对 fastcgi 请求做处理。处理流程为：暂停1s，打印处理的进程id。(为了能看到不同进程做了响应，因此对进程暂停1s处理，并打印进程id。)

```go
// fastcgi.go
package main

import (
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
	"strconv"
	"time"
)

type FastCGIServer struct{}

// 暂停1s， 打印标识的进程id
func (s FastCGIServer) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	time.Sleep(time.Second)
	resp.Write([]byte("ProcessId: " + strconv.Itoa(os.Getpid()) + "\n"))
}

func main() {
	listener, _ := net.Listen("tcp", "127.0.0.1:9001")
	srv := new(FastCGIServer)
	fcgi.Serve(listener, srv)
}
```

通过如下命令得到一个简单的fastcgi 二进制文件。通过监听127.0.0.1:9001 端口做fastcgi 处理。处理内容为暂停1s，并打印处理的进程id。(为了能看到不同进程做了响应，因此对进程暂停1s处理，并打印进程id。)

```
go build -o fastcgi fastcgi.go
```

生成的fastcgi 就是一个简单的fastcgi 服务。功能为暂停1s，并输出当前进程的进程ID。

#### 修改 supervisor 的配置

修改supervisor 的配置，将fastcgi 服务添加到supervisor 管理，并启动6个fastcgi 进程。


在supervisord.conf 添加如下配置：

```
[fcgi-program:fastcgi_test]
socket=tcp://127.0.0.1:9001
command=/root/test/fastcgi             
autostart=true
stopwaitsecs=1000
autorestart=true
user=root
process_name=%(program_name)s_%(process_num)02d
numprocs=6
```

修改完成后，需要刷新supervisord 的配置，并启动fastcgi。

```
supervisorctl update 
supervisorctl start fastcgi_test:*     #  因为启动的fastcgi 有多个，因此需要加 :*
```

#### 修改nginx 的配置

Nginx 配置如下：

```
	server {
	        listen 127.0.0.1:8080;
	        location / {
	                include         fastcgi.conf;
	                fastcgi_pass    127.0.0.1:9001;
	        }
	}
```

并通过如下命令重新加载 nginx 配置。

```
nginx -s reload
```

#### 做一个简单的请求实验

对nginx 重新加载配置后，我们请求8080 端口，看服务的请求情况：

post 10次请求：

```
#  for i in `seq 1 10`; do curl 'http://127.0.0.1:8080/app?helloworld' & done
#  ProcessId: 11319ProcessId: 11299ProcessId: 11300ProcessId: 11307ProcessId: 11307ProcessId: 11311ProcessId: 11311ProcessId: 11315ProcessId: 11315ProcessId: 11319
```

返回结果，processId 被均匀的分到不同的fastcgi 上。

当某个 fastcgi\_test 意外退出时，supervisor 可以再次启动一个fastcgi\_test 做补充，这就实现了PHP-FPM master 进程的主要功能。

### 实现原理

我们知道，正常情况下，一个端口只能被一个进程监听。但是刚刚看到的情况是，多个fastcgi同时启动，监听 9001 端口。这是因为linux 系统中，如果父进程监听端口后，fork 的子进程可以继承父进程的文件描述符，因此多个进程可以监听同一个端口。
通过pstree 命令我们可以看到：
![supervisor 启动的多个fastcgi 监听同一个端口](supervisor_fastcgi.png)

### 实现的功能

supervisor 在管理fastcgi 的进程中，和管理普通进程的差别是，supervisord 进程会创建socket 链接，共享给 supervisor fork 的fastcgi 进程，但是非fastcgi 的进程不会被共享。

![](/images/weixin_logo.png)
