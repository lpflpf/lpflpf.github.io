---
layout: post
title: supervisor 的使用和进阶 (1)
subtitle: 路漫漫其修远兮，吾将上下而求索。
date: 2020-04-03 16:30:42
author: 李朋飞
tags:
  - 服务管理

---

再也不怕进程意外退出。

<!--more-->

> supervisor 是Python 开发的一套通用的进程管理程序，用于管理类Unix系统上的应用程序。
> 可以实现对服务的命令行、WEB、XML等方式的管理，实现对服务的启动、重启、关闭等操作。

## supervisor 可以干什么

  - 管理进程，对进程进行开启、关闭、重启等服务；
  - 守护管理的进程。当进程关闭后，可以自动重启；
  - 管理一组进程，一组进程同时启动，关闭，重启等服务；
  - 提供事件管理，用于管理的进程触发的事件进行报警的功能（supervisor 3.0 引入）；
  - 提供了对监听同一个unix socket 文件的cgi服务的管理；
  - 提供web服务做服务管理；
  - 提供XMLPRC服务做二次开发；

## 服务安装

以生产环境使用较多的CentOS 为例, 使用yum包管理器即可完成安装，命令如下：

```
yum install supervisor

```

如何非centos系统，则可以使用Python 强大的包管理器pip来完成安装。 

```
pip install supervisor
```

安装完成后，生成配置文件。supervisor 提供了 echo\_supervisord\_conf 命令，用于生成supervisord 的配置文件。
*如果pip安装，echo\_supervisord\_conf 会安装在相应pip的目录下。*

一般配置文件会保存至/etc/ 目录下，生成方式如下：

```shell
echo_supervisord_conf > /etc/supervisord.conf
mkdir -p /etc/supervisor.d   // supervisord 支持include 的方式将多个配置放置不同文件中, 需要配置文件中指定
```

supervisor 提供了两个命令给用户：
    - supervisord   supervisor 守护其他服务的进程
    - supervisorctl supervisor的命令行工具

最后启动supervisor即可：
```shell
supervisord -c /etc/supervisord.conf
```

**PS:** 

如果使用yum管理安装的，可以直接使用systemctl 管理启动和暂停supervisord。

## 服务的配置

简单介绍两种比较常用的配置。

### 对简单进程的管理

```
;[program:theprogramname]
;command=/bin/cat              ; 启动命令
;process_name=%(program_name)s ; process_name expr (default %(program_name)s)
;numprocs=1                    ; 同一个任务如果需要启动多次，需要配置此项，并配置process_name为类似于 %(program_name)%02d 格式
;directory=/tmp                ; 任务执行的当前目录
;umask=022                     ; 进程文件权限掩码 （默认创建文件为0644
;priority=999                  ; 优先级
;autostart=true                ; 是否自动开启。（当supervisord 启动时）
;startsecs=1                   ; 程序开启 startsec s内不退出
;startretries=3                ; 最大尝试次数
;autorestart=unexpected        ; 是否退出后自动重启 （默认不重启，对于经常意外退出的服务可以开启）
;exitcodes=0                   ; 判断是否正常退出码
;stopsignal=QUIT               ; 关闭的信号 （默认时TERM， 也就是Ctrl-C）
;stopwaitsecs=10               ; 服务关闭等待事件。若关闭事件超出，则发送 SIGKILL 信号 （也就是 kill -9)
;stopasgroup=false             ; 是否为杀死子进程，默认不杀死。（将会出现未纳入管理的孤儿进程）
;killasgroup=false             ; 发送SIGKILL 信号的时候，是否杀死子进程
;user=chrism                   ; 启动的用户
;redirect_stderr=true          ; 将进程的标准输出重定向为标准输出
;stdout_logfile=/a/path        ; 进程标准输出
;stdout_logfile_maxbytes=1MB   ; 之日滚动大小，默认50M
;stdout_logfile_backups=10     ; 日志最多保留个数
;stdout_capture_maxbytes=1MB   ; 捕获输出的日志，当事件开启时发送给event_listener （也就是事件监听进程）
;stdout_events_enabled=false   ; 对于标准输出的情况是否发送event
;stdout_syslog=false           ; 是否发送到syslog
;stderr_logfile=/a/path        ; 标准错误输出路径
;stderr_logfile_maxbytes=1MB   ; 最大标准错误输出的日志文件大小（默认50M）
;stderr_logfile_backups=10     ; 日志最多保留个数 （10个） 
;stderr_capture_maxbytes=1MB   ; 捕获标准错误输出的日志，当 stderr_events_enabled 开启时发送给event_listener
;stderr_events_enabled=false   ; 对于标准错误输出的情况是否发送event
;stderr_syslog=false           ; 是否发送错误输出日志到syslog
;environment=A="1",B="2"       ; 子进程的环境变量设置，可用的变量有 `group_name`, `host_node_name`, `process_num`, `program_name`, `here`。

```

### 对进程组的管理

除了对单个进程（或者相同的进程）进行控制外，还可以将多个program分组进行控制。

例如有服务 bar，baz, 可以定义进程组：

### 其他模式

如cgi 服务管理、事件监听，后面做详细讨论。

```
[group:foo]
programs:bar, baz
priority:999
```

对group做开启，暂停，则对下面的bar，baz都会生效。使用时可以用如下命令:

```shell
supervisorctl [start | stop | restart | status] foo:
```

当然，可以使用通过 foo:bar 管理bar服务


## 常见命令行

supervisorctl 是supervisor 提供的配套命令行工作，用于对supervisor做命令行控制。

### 本地服务的启动和暂停

本地服务的启动暂停，使用的是 unix socket的方式对supervosrd 发送命令的。因此，使用本机操作命令，必须指定unix socket 的路径。
配置如下：

```
[supervisorctl]
serverurl=unix:///var/tmp/supervisor.sock
```

命令行操作服务的启动和暂停

```shell
supervisorctl [start | stop | restart | status ] jobname

```

### 远程的启动和暂停

如果开启了远程操作的端口，也可以通过命令行方式操作远程服务。

```shell
supervisorctl -s hostname:9001 [-u user] [ -p password] [ start | stop | restart | status ] jobname
```

### supervisor 配置更新和修改

- supervisor 服务配置更新， 并对修改的服务做相应操作

```shell
supervisorctl update

```

- supervisor 服务配置更新，并重启所有服务。

```shell
supervisorctl reload
```

其他操作可以参考[supervisor官方文档](http://www.supervisord.org/running.html#supervisorctl-command-line-options)。

## Web 服务

supervisor 提供了简约而不简单的操作见面，可以在浏览器端对服务做远程控制。

web 服务需要做如下配置，开启服务监听

```
[inet_http_server] 
port=*:9001   
username= test       
password=testpass   ; 可以不需要账号密码
```

以下为操作界面：
![supervisor web 服务](supervisor_1.jpg)

web 服务除了可以对服务做启动暂停等操作外，还可以远程查看应用的日志，监控服务的log 是否正常。这在普通的web 服务中还是比较常用的。

## 二次开发

supervisor 提供了XMLRPC接口用于使用它的人可以二次开发利用。

例如，可以通过远程访问 supervisor 服务控制应用服务的启动、暂停。获取应用服务当前的服务状态等。因此可以**通过supervisor 的xmlrpc 监控对supervisor 管理的服务做多服务远程监控**

使用xmlrpc时，需要设置inet\_http\_server, 用于监听rpc和web服务的端口。(建议仅监听内网IP，并设置相应密码)
对于PHP服务，我做了简单的封装,可以从Github中获取：[github.com/lpflpf/supervisor\_phpctl](https://github.com/lpflpf/supervisor_phpctl)

```php

function monitor($host, $port, $jobname){
    $server = new Supervisord($host, $port);

    $state = $server->getState();

    switch ($state['statename']){
        case 'RUNNING':    // 服务正常
            break;       
        case 'RESTARTING': // 服务重启
            break;
        case 'SHUTDOWN':   // 服务关闭
            break;
        case 'FATAL':      // 服务出现错误退出
            //alarm();
            return; 
    }
}
```

## supervisor 原理

- supervisord 管理的任务进程都是supervisord 的子进程, 通过 fork/exec 方式启动子进程。
- supervisord 杀死子进程，其实就是发送给子进程一个中断信号（这个信号可以自定义, 参数为stopsignal， 默认为TERM信号）

## 其他需要强调的点

- supervisor 不会随着系统的重启而启动，因此那些依赖supervisor的服务也不会随着系统重启而启动。(别问我是怎么知道的)
解决办法也简单。只需要将supervisor 开机启动就行。不同版本操作系统不太一样。

centos 可以用如下方法：

```shell
chkconfig --add supervisord
chkconfig supervisord on
```

- supervisor 管理的进程可能存在多种状态，在做服务监控时需要注意, 如下为进程状态转移图：

![supervisor进程转移图](subprocess-transitions.jpg)

  **需要注意backoff 状态，当服务不断进行快速关闭重启，则会进入baockoff 状态。这种状态一般也是有问题的。**

- 对于进程组的操作

    - 如果操作进程组中的某个进程，jobname 使用自定义的process\_name。
    - 如果操作进程组中的所有进程，使用process\_name:\* 即可 

##  其他类似服务

  - [runit](http://smarden.org/runit/)
  - [launchd](https://en.wikipedia.org/wiki/Launchd)
  - [daemontools](http://cr.yp.to/daemontools.html)
  - systemctl

![](/images/weixin_logo.png)
