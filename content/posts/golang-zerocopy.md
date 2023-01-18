---
title: Golang 中构建零拷贝的文件Web服务器
date: 2021-12-03 17:37:40
tags:
  - golang
---

本文讲从Golang 的文件服务器说起，接着探究sendfile 系统调用是什么，最后总结下零拷贝的使用场景。

<!--more-->

## 构建一个文件服务器

在Golang 中，如何构建一个零拷贝的文件服务器呢，如下是全部代码:

```golang
package main

import "net/http"

func main() {
        // 绑定一个handler
        http.Handle("/", http.StripPrefix("/static/", http.FileServer(http.Dir("./output"))))
        // 监听服务
        http.ListenAndServe(":8000", nil)
}
```

嗯，没有看错。两行代码实现了一个文件服务器。

### FileServer 处理Handler 如何实现？ 

对于处理文件请求的Handler，按照我们的想法，实现将会非常简单：判断文件类型：如果请求的是目录，则返回目录列表;如果请求的是文件; 则 io.Copy 直接返回数据。
但真正的实现比我想象中要略复杂。

通过跟踪代码，画出了如下的简易流程图：

[!ServeFile](servefile.png)

在最后一个步骤tcp Write，即将数据写入到tcp流中。serveFile 使用的是 `io.CopyN(w, sendContent, sendSize)`
当代码看到这里，自我感觉很满意。因为实现貌似和我们想象中没有太大出入。

接着，看看io.CopyN 方法:
```golang
func CopyN(dst Writer, src Reader, n int64) (written int64, err error) {
	written, err = Copy(dst, LimitReader(src, n)) 
	if written == n {
		return n, nil
	}
	if written < n && err == nil {
		// src stopped early; must have been EOF.
		err = EOF
	}
	return
}
```

仅仅是加了限定的 io.Copy 方法。 在看看我们熟悉的io.Copy 方法.

```golang
func copyBuffer(dst Writer, src Reader, buf []byte) (written int64, err error) {
	if wt, ok := src.(WriterTo); ok {
		return wt.WriteTo(dst)
	}
	if rt, ok := dst.(ReaderFrom); ok {
		return rt.ReadFrom(src)
	}
    // 省略
    // 创建buffer
    // for {
    //    read
    //    write
    // }
}
```

src 读出来，写进dst。一起都按照我们的想法来的。没毛病。

**等等， 好像ReaderFrom 接口在哪里见过？**

```golang
type ReaderFrom interface {
	ReadFrom(r Reader) (n int64, err error)
}
```

```golang
func (c *TCPConn) ReadFrom(r io.Reader) (int64, error) {
	if !c.ok() {
		return 0, syscall.EINVAL
	}
	n, err := c.readFrom(r)
	if err != nil && err != io.EOF {
		err = &OpError{Op: "readfrom", Net: c.fd.net, Source: c.fd.laddr, Addr: c.fd.raddr, Err: err}
	}
	return n, err
}
```

tcp 链接实现了ReadFrom 接口。这个实现到底干了什么？

```golang

func (c *TCPConn) readFrom(r io.Reader) (int64, error) {
	if n, err, handled := splice(c.fd, r); handled {
		return n, err
	}
	if n, err, handled := sendFile(c.fd, r); handled {
		return n, err
	}
	return genericReadFrom(c, r)
}

```

如果编译的linux binary，则会在splice 方法中调用系统调用splice.
调用了sendfile，内部实现则是调用了系统调用方法Sendfile。

## sendfile有啥特别的？read/Write 不香吗？

从man 中找到了答案。

```
 sendfile()  copies  data between one file descriptor and another.  Because this copying is done within the kernel,
 sendfile() is more efficient than the combination of read(2) and write(2), which would require  transferring  data
 to and from user space.
```

sendfile 用于两个文件描述符之间的数据拷贝，由于是内核态上做的数据操作，避免了内核缓冲区和用户缓冲区的数据拷贝，所以被称为零拷贝技术。效率上要比需要做缓冲过去拷贝的 read/write 方法效率高很多。

### 工作原理

![read-write](read-write.png)
![sendfile](sendfile.png)
![sendfile-sgdma](sendfile-sgdma.png)


## 注意事项

1. sendfile 必须是一个支持mmap 函数的文件描述符; 目标fd必须是socket
