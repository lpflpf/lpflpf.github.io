---
title: go mod
date: 2019-06-20 18:21:43
tags:
    - go build
category:
    - golang
---

go mod 工具简单入门介绍。
<!--more-->

### 简介

目前 Golang 项目的包管理方式如下：
1. 裸奔模式
  - 配置项目目录为GOPATH路径
  - 将依赖项目放在 src/... 目录下
2. dep 工具 (类似的有godep, govendor 工具)
  - dep 工具为官方工具
  - 将在目录下创建vendor 目录，依赖下载至vendor 目录下
3. mod 工具
  - Go 1.11 版本以后自带子命令
  - 去掉GOPATH依赖

### 是否需要手动开启GO Module 模式

> GO111MODULE 的取值为 off, on, or auto.

- **off**: GOPATH mode，查找vendor和GOPATH目录
- **on**：module-aware mode，使用 go module，忽略GOPATH目录
- **auto**：如果当前目录不在$GOPATH 并且 当前目录（或者父目录）下有go.mod文件，则使用 GO111MODULE， 否则仍旧使用 GOPATH mode。

### 如何使用

- step 1:  **初始化mod, 将go.mod 文件，保存当前目录的pkg name**  
```shell
    # go mod init current_pkg_name
```
- step 2:  **将项目依赖的多有pkg下载；若GOPATH为空，则放置~/go 目录下，否则放置到GOPATH 目录下**  
```shell
	# go mod tidy
```
- step 3:  
```shell
	# go build pkg_name
```

> 其他自命令  go mod [download | edit | graph | vendor | verify | why]
