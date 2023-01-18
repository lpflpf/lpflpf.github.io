---
title: golang 1.13 
date: 2019-09-19 15:15:00
tags:
    - golang
category:
    - golang
---


2019.09.03， golang 发布了新版本，一起来学习下本次修改的内容。

<!-- more  -->

### 二进制数字标识

使用 0b 或者 0B 标识二进制数字。 例如： 0b1011 标识11

### 八进制数字标识

使用 0o 或者 0O 标识8 进制整数。例如： 0o660, 目前使用的包含前导0的数字仍旧是合法的。


### 十六进制浮点数标识

使用 0x 或者 0X 是使用十六进制标识浮点数。其中指数是在p标识后面，是2的指数倍。例如 0x1.0p-1021 ，代表 2 ^ -1021 次方

### 虚数的数字标识

虚数虚部的标识已支持已有的所有表达方式，例如:

0i
0123i         // == 123i for backward-compatibility
0o123i        // == 0o123 * 1i == 83i
0xabci        // == 0xabc * 1i == 2748i
0.i
2.71828i
1.e+0i
6.67428e-11i
1E6i
.25i
.12345E+5i
0x1p-2i       // == 0x1p-2 * 1i == 0.25i

### 数字分割符

按照国外按下划线分为多个组, 下划线可以出现在仍以两个数字或者数字前缀和首个数字之间如: 1\_000\_000, 0b\_1010\_0110, 3.1415\_9265


### 移位运算符不再需要uint 变量

减少了不必要的uint 操作

### 工具的修改

  golang 自带了很多工具命令，每次版本更迭时，可能会做相关工具的更新。

#### module

  - GO111MODULE 环境继续默认值为auto, 但是auto 默认无论当前目录或者子目录包含go.mod 文件（即使当前目录在GOPATH/src 目录下）均认为开启gomodule。这个修改将简化现有使用GOPATH 的代码和使用gomodule但使用方是使用GOPATH的代码维护。
  - 新增GOPRIVATE 环境变量, 用于标识非共有仓库的数据源
  - 设置GOPROXY 控制代理
  - GOSUMDB 环境变量标识, 用于验证包的有效性的地址 , 默认为 sum.golang.org/lookup/xxx， 关闭方式  "go env -w GOSUMDB=off"
  - 修改后， `go get -u` 仅下载当前目录下的依赖包，如果更新所有的依赖包，需要使用 `go get -u all`
  - go get 不再支持 `-m`. 

#### Go Command

  - `go env -w / -u` 设置或者删除用户的环境变量值, 环境变量值将被存在 os.UserConfigDir()
  - `go version [-m] [-v] [file ...]`  若指定了file，则打印响应可执行文件的所使用的go版本， 若使用 `-m` , 打印嵌入没款的版本信息。如果是一个目录，将打印目录包含的可执行文件的信息和相关子目录的可执行文件的go编译版本。
  - go build flag 变更
    - -trimpath 移除所有编译自带的文件系统链接路径，减少编译依赖。 (这个对于服务跨版本迁移相当有帮助)
    - -o 如果传入的是一个已存在的目录，go build 生成的可执行文件将写入此目录中。
    - -tags 建议使用逗号分割编译标识符,当然空格也在维护，但已经标识为**即将废弃**。

#### Compiler toolchain

  - 编译器做了优化，对于逃逸分析更加精准。当然如果需要做回归分析，可以使用 -gcflags=all=-newscape=false 做老的逃逸分析。

#### Assembler

  在 ARM v8.1 上增加了多个原子指令

#### gofmt

  主要对于 数字样式变更的修改。

#### godoc

  godoc 不再在golang 包中出现，需要通过 `go get golang.org/x/tools/cmd/godoc` 安装

### runtime

  - **defer 性能提升 30%**

### Core Library

  - TLS 1.3 支持。 在`crypt/tls` 包中，默认支持 TLS 1.3
  - crypto/ed25519 `golang.org/x/crypt/ed25519`包迁移至 `crypto/ed25519` 中。该包是 Ed25519 数字签名算法的实现。
  - **Error wrapping** golang 支持错误包装。
    - 一个错误 e 可以包含另外一个错误w. e 通过调用 Unwrap 方法可以拿到错误w
    - fmt.Errorf 可以通过 %w 创建一个wrapped 错误
    - errors 包提供了 errors.Unwrap, errors.Is errors.As 三个方法用于错误包含的判定。
  - 其他类库的小修改
    - `bytes.ToValidUTF8()` 方法。 替换不合法的u8 编码为指定的字符。
    - `context
    - `crypto/tls`包中，sslv3 再1.13 标记为即将废弃，在 go1.14 将被移除
    - `crypto/x509` 
    - `database/sql` NullTime 类型代表可能为null 的time.Time；NullInt32 代表 可能为null 的 int32 类型
    - `debug/dwarf` 
    - `errors` 添加 As, Is, UnWrap 方法
    - fmt 
      - "%x %X" 支持 浮点数和复数的16进制格式化
      - "%0" 输出带有前导0o的8进制数
      - Errorf 添加 "%w", 用于生成错误包装函数
    - go/scanner
    - go/types
    - html/template
    - log
    - math/big
      - Rat 增加了函数 Rat.SetUint64(), Rat.SetString 支持非十进制浮点数表达
    - net
    - net/http
    - os
      - 新的UserConfigDir 方法返回用户配置目录的根目录
      - 如果File 通过 O_APPEND 标识打开， WriteAt 方法不可用，将返回错误。
    - os/exec
      - windows 中，Cmd 的环境变量线性继承自 %SYSTEMROOT% 的值，除非Cmd.Env 显性赋值。
    - reflect
      - 新增 Value.IsZero 方法，判断是否为0值
      - MakeFunc 允许在返回值的类型上做复制转换。有利于那种定义了抽象返回类型，而实现是一个具体返回值的方法调用
    - runtime
    - strconv
    - strings
    - sync
      - 通过编译优化，将 Mutex.Lock, Mutex.Unlock, RWMutex.Lock, RWMutex.RUnlock, Once.Do 编译优化[inlining](https://github.com/golang/go/wiki/CompilerOptimizations#function-inlining) 化。对于amd64上无竞争的互斥量, Once.Do 快了一倍， Mutex/RWMutex 快10%
    - syncall
    - syscall/js
    - testing
    - text/scanner
    - text/template
    - time
    - unicode

> [go 1.13 release note ](https://golang.google.cn/doc/go1.13) 
