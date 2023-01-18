---
title: golang 二进制文件中添加编译信息
date: 2019-06-20 16:34:38
tags:
  - golang
  - go build
category:
  - golang
---

在编译二进制程序时，动态赋值程序的某些值，使程序包含了可靠的编译信息。
<!--more-->

### Go 二进制中包含编译信息
  - 如果服务上线后，不知道此二进制文件是哪个版本产出的二进制，那么本文可以帮助你实现相关的功能。
  - 在二进制代码发布时，传入必要的版本信息，以便日后可查看相关信息。
  - 可用于 git-runner 中，直接获取版本信息、分支信息等，填充相应参数。

#### 效果展示
`binary` 是我们例子中的二进制文件

```shell
    # ./main -v
    Commit ID  : 123
    Build  Name: version test
    Build  Time: 20190620
    Build  Vers: 1.1
    Golang Vers: go version go1.10.3 linux/amd64
```

#### 实现方法
1. 在golang 解析参数部分添加如下内容:

```go
package main

import "github.com/lpflpf/version"
import "flag"

func main() {
	var showVer bool
	// 为了举例，所以仅使用了-v 选项
	flag.BoolVar(&showVer, "v", false, "show build version")
	flag.Parse()

	if showVer {
		version.Show()
	}
}
```

version 包如下：

```go
package version

import (
    "fmt"
    "os"
)

// 连接过程中修改如下5个参数，可以自行添加使用
var (
    BuildVersion string
    BuildTime    string
    BuildName    string
    CommitID     string
    GoVersion    string
)

func Show() {
    fmt.Printf("Commit ID  : %s\n", CommitID)
    fmt.Printf("Build  Name: %s\n", BuildName)
    fmt.Printf("Build  Time: %s\n", BuildTime)
    fmt.Printf("Build  Vers: %s\n", BuildVersion)
    fmt.Printf("Golang Vers: %s\n", GoVersion)
    os.Exit(0)
}
```

2. 编译程序，编译脚本如下：

```
BUILD_TIME=`date +%Y%m%d`
BUILD_VERSION=1.1
COMMIT_ID=123
GO_VERSION=`go version`
BUILD_NAME="version test"
VERSION_PKG='github.com/lpflpf/version'
LD_FLAGS="-s -w -X '$VERSION_PKG.BuildTime=$BUILD_TIME'                \
                         -X '$VERSION_PKG.BuildVersion=$BUILD_VERSION' \
                         -X '$VERSION_PKG.BuildName=$BUILD_NAME'       \
                         -X '$VERSION_PKG.CommitID=$COMMIT_ID'         \
                         -X '$VERSION_PKG.GoVersion=$GO_VERSION'"
go build -ldflags "$LD_FLAGS" main.go
```

3. 执行脚本， 则看到本文开头说明的二进制版本信息



### 原理分析
在golang 进行连接包时，允许将字符串传入包的变量中。因此，在编译时，通过ld选项添加相应变量，实现了二进制中保存编译信息的功能。


#### 参考文档
1. [golang 连接说明](https://golang.org/cmd/link/)
2. [version 包](https://github.com/lpflpf/version/)

![](/images/weixin_logo.png)
