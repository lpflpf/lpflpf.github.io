---
title: golang 性能测试 (1)
p: golang-benchmark
date: 2020-04-09 10:04:12
tags:
  - golang
  - benchmark

category: 
  - golang
---

本文介绍golang 如何做基准性能测试。

<!--more-->

编写完代码除了跑必要的单元测试外，还需要考虑代码跑起来的性能如何。性能的衡量其实就是程序运行时候进程的内存分配，CPU消耗情况。

golang 语言在提供了功能测试的基础上，提供了丰富的性能测试功能。

### SHOW CODE

首先，从一个例子来讲起。 随便写一个简单的快速排序，然后和系统自带的排序做一个性能比较。

如下为简版快排的代码：

```go
package benchmark

import "sort"

func QSort(data []int) {
	myqsort(data, 0, len(data)-1)
}

func myqsort(data []int, s, e int) {
	if s >= e {
		return
	}

	t := data[s]
	i, j := s, e

	for i < j {
		for ; i < j && data[j] >= t; j-- { }
		for ; i < j && data[i] < t; i++ { }
		if i < j { break }

		data[i], data[j] = data[j], data[i]
		i++
		j--
	}

	data[i] = t
	myqsort(data, s, i-1)
	myqsort(data, i+1, e)
}

```

然后编写一个测试的test。

```go
package benchmark

import "testing"
import "math/rand"
import "time"
import "sort"

var ints []int

// 长度为 1w 的数据使用系统自带排序
func BenchmarkSort10k(t *testing.B) {
	slice := ints[0:10000]
	t.ResetTimer()   // 只考虑下面代码的运行事件，所以重置计时器
	for i := 0; i < t.N; i++ {
		sort.Ints(slice)
	}
}

// 长度为 100 的数据使用系统自带排序
func BenchmarkSort100(t *testing.B) {
	slice := ints[0:100]
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		sort.Ints(slice)
	}
}

// 长度为 1w 的数据使用上述代码排序
func BenchmarkQsort10k(t *testing.B) {
	slice := ints[0:10000]
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		QSort(slice)
	}
}

// 长度为 100 的数据使用上述代码排序
func BenchmarkQsort100(t *testing.B) {
	slice := ints[0:100]
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		QSort(slice)
	}
}

// 数据初始化，为了保证每次数据都是一致的。
func TestMain(m *testing.M) {
	rand.Seed(time.Now().Unix())
	ints = make([]int, 10000)

	for i := 0; i < 10000; i++ {
		ints[i] = rand.Int()
	}

	m.Run()
}
```

运行命令 ：

```
# go test -cover -count 3  -benchmem  -bench=.
```

运行结果如下图：

![](benchmark.jpg)

基准测试，默认将每个方法执行1s中，然后展示执行的次数，每一次执行的耗时, 上述还展示了内存每次分配的大小，以及每次benchmark分配的次数。上述的命令行指定了运行次数为3次，显示代码覆盖率和内存分配情况。

从基准测试的结果可以分析出：对于1w数据量的排序，自带的排序比我的排序算法要快20倍左右；100数据量的排序，手撸的排序略胜一筹。
从内存分析来讲，系统自带的会使用4B的数据，而我的算法无内存分配。

### INTRODUCE BENCHMARK

引入golang 提供的 `testing` 包，写需要的基准测试的方法（方法名必须以Benchmark开头, 参数必须为 \*testing.B）。

若需要做一些数据初始化的工作，可以如上写一个TestMain 方法，将数据初始化的工作在这里完成。

除了这些，可以看\*testing.B, \*testing.M 的相关方法即可。 

最后，只要运行官方提供的 `go test -bench=.` 命令,即可开始跑基准测试。 当然，还有其他选项可以满足我们多样的需求。
例如：
  - \-cpu 1,2,4 指定运行的cpu 格式  
  - \-count n   指定运行的次数
  - \-benchtime 每一条测试执行的时间 （默认是1s）
  - \-bench     指定执行bench的方法， `.` 是全部
  - \-benchmem  显示内存分配情况
  
其他参数可以通过 `go help testflag` 查看


### WHY SO SLOW

  1. 我这里选取的是第一个数作为中位数，数据越大越可能出现倾斜，排序慢的概率也大。
  2. 正常的排序包中，都会在对小于等于12 个数的数组做排序时使用希尔排序，速度也有很大提升。

> 除了简单的做性能测试外，golang 还自带了性能分析的工具，我们可以快速找出代码中的内存分配、cpu消耗的核心区，帮助我们解决服务的性能问题。下篇文章将做详细了解。

![](/images/weixin_logo.png)
