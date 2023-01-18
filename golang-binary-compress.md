---
title: go binary compress
date: 2019-06-20 17:59:20
tags: 
  - go build
  - golang
category: golang
---

golang 压缩的方式: 1. build 添加去除调试标识; 2. 使用upx 工具。
<!--more-->

### go 二进制文件压缩
> 由于git中保存二进制文件，可能会使项目过大，可以将二进制文件压缩，使程序更加便携。

### 去掉 gdb 调试信息和符号表

```shell
  # go build -ldflags " -s -w"
```

 - s 去掉符号表信息
 - w 去掉调试信息

### 使用upx 工具压缩

 - 可压缩 50% - 70% 大小
 - 原理： 包含自解压程序，类似exe 文件
 - 编译机器安装upx 命令;部署环境不需要安装
 - 命令如下： （可以添加参数是文件压缩更小）


 ```shell
  # upx binary_filename
 ```

#### 工具连接


1. [upx](https://upx.github.io/)
