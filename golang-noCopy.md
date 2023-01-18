---
title: 为什么需要 noCopy
date: 2019-07-05 11:48:05
tags:
  - noCopy
category:
  - golang
  - php
---

本文对golang 的noCopy 机制做简要分析。

<!--more-->
### php 的noCopy 的实现

php 中对象赋值是浅拷贝，即赋值都仅仅是copy了一次指向对象的指针而已。因此，php实现的noCopy 是针对深拷贝而言的。

深拷贝是使用clone 关键字实现的。

如果需要实现php的noCopy，只需要将php的魔术方法`__clone` 设置为私有即可。

#### 代码示例

```php
<?php

class Copy
{
    public $property = 1;
}

class noCopy 
{
    public $property = 1;

    private function __clone(){}
}

$data = new Copy();
echo $data->property . "\n";   // 1
$ref = $data;
$copy = clone $data;
$data->property = 2;
echo $ref->property . "\t" . $copy->property . "\n";    //1 ,2

$data = new noCopy(); 
echo $data->property;   // 1
$ref = $data;
$copy = clone $data;  //   Call to private noCopy::__clone() from context ...
$data->property = 2;
echo $ref->property;
```

对于一个互斥量mutex, 其本质是包含有一定状态的变量。如果一个对象持有mutex，且该对象通过mutex,操作持有的资源.

### 为什么需要nocopy 呢？
对于一个**互斥锁**，实现是一个int值 和一个uint值构成的结构体。两个值标识了锁的状态。
如果锁可以copy,那锁状态也将被copy(由于struct 是值拷贝的)，当锁状态再次更新后，copy后的值将不再有效。
因此，对于实现了`sync.Locker`接口的类型来说，理论上其实例是不能再次被赋值的。

### golang noCopy 的实现
由于golang 中struct对象赋值是值拷贝，没有php类中的魔术方法。[golang issue](https://golang.org/issues/8005#issuecomment-190753527)里面，
golang sync 包中:
	- `sync.Cond`
	- `sync.Pool`
	- `sync.WaitGroup`
    - `sync.Mutex`
    - `sync.RWMutex`
    - …… 
 禁止拷贝，实现方式采用noCopy 的方式。

```go
package main

import "fmt"

type noCopy struct{}

// Lock is a no-op used by -copylocks checker from `go vet`.
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

type S struct {
	noCopy
	data int
}

func main() {
	var s S

	ss := s
	fmt.Println(ss)
}
```

golang 没有禁止对实现`sync.Locker`接口的对象实例赋值进行报错，只是在使用go vet 做静态语法分析时，会提示错误。

```
# command-line-arguments
./nocopy.go:19: assignment copies lock value to ss: main.S
./nocopy.go:20: call of fmt.Println copies lock value: main.S
```
> [golang Issue](https://golang.org/issues/8005#issuecomment-190753527)
