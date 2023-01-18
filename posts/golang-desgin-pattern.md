---
title: golang 中的一些设计模式
date: 2019-11-04 17:33:05
tags:
  - golang
  - 设计模式
---

无论什么代码写多了，都会发现有很多套路在里面，坦白的说，那可能就是一种设计模式了，今天也总结一两点 `Golang` 中常用的设计模式。

<!--more-->

## 策略模式

### 策略模式的简单定义
一个服务定义一个抽象的接口，而接口可以有多种实现方式，在使用过程中，服务可以对不同的实现做替换。

操作系统中，打开一个.go 文件，可能有很多方式。vscode，sublime，vim等，我们也可以设置默认的打开方式。抽象来看，这些也可以看作是各种策略，可以指定策略来完成我们自定义的操作。 而前提是系统给我们提供了通用的接口，让我们来实现这些策略。

### 一个简单栗子
写过golang的同学一般都会操作数据库，如果要使用 `mysql` 作为数据源，可能代码需要引入 Golang 的一个包数据驱动包，例子如下：

```go

import  _ "github.com/go-sql-driver/mysql"
import  "database/sql"

func doSomething(){
    if db, err := sql.Open("mysql", dsn); err == nil {
        // do something
    }
}

```

但是import 的时候，程序做了什么，使得驱动得以注册。
我们可以从两个地方找到答案，一个是 mysql 的驱动包，一个是 golang 官方的 `database/sql` 包。

首先，我们可以从`database/sql` 包看起，官方包中提供了三个方法，用于获取或者操作驱动：
要注册一个驱动，需要实现 `database/sql/driver` 包下 Driver 的接口。

```go

// 注册驱动
func Register(name string, driver driver.Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("sql: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("sql: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// 删除所有注册的驱动
func unregisterAllDrivers() {
	driversMu.Lock()
	defer driversMu.Unlock()
	// For tests.
	drivers = make(map[string]driver.Driver)
}

// 当前注册了哪些驱动
// Drivers returns a sorted list of the names of the registered drivers.
func Drivers() []string {
	driversMu.RLock()
	defer driversMu.RUnlock()
	var list []string
	for name := range drivers {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}
```

而在MySQL的驱动包中， 选择`github.com/go-sql-driver/mysql@v1.4.1` 包作为例子， driver.go 文件中, 最后有这么一段启动代码：

```go
func init() {
    sql.Register("mysql", &MySQLDriver{})
}
```

从实现中可以看出 `*MySQLDriver` 便是一种Driver 的一个实现而已。
因此，驱动就在直接引入mysql包的时候，悄无声息的被注册到了sql的私有变量 drivers 中了。当引用该驱动时，直接可以读取drivers中相应的驱动方法。
> 当然，抽象考虑的话，这就是一个策略模式的实现。提供了注册策略的接口，当数据源切换时，可以任意切换相应的驱动（策略）。

如果代码看的不尽兴，我们可以再看一个易懂的例子。

### 另一个简单的例子

Kafka 是一个非常经典的消息队列，Kafka消费者可以按照消费组的方式进行消费，当多个客户端按照同一个消费组消费消费同一个主题（Topic）的消息时，需要按照一定的策略将客户端与Partition的对应关系协调好，这样多个客户端才能正常消费，这就是Consumer Group 的Reblance。

`github.com/shopify/sarama` 是golang 实现的kafka 客户端的一个比较常用的包。 包中balance\_strategy.go文件中是分配策略的一些实现。

其中，分配策略接口定义如下：

```go
type BalanceStrategy interface {
	// Name uniquely identifies the strategy.
	Name() string

	// Plan accepts a map of `memberID -> metadata` and a map of `topic -> partitions`
	// and returns a distribution plan.
	Plan(members map[string]ConsumerGroupMemberMetadata, topics map[string][]int32) (BalanceStrategyPlan, error)
}
```

而 `balanceStrategy` 是该接口的一个简单实现，而这里又实例化了两种策略：

- `BalanceStrategyRange`  

```go
func(plan BalanceStrategyPlan, memberIDs []string, topic string, partitions []int32) {
	step := float64(len(partitions)) / float64(len(memberIDs))

	for i, memberID := range memberIDs {
		pos := float64(i)
		min := int(math.Floor(pos*step + 0.5))
		max := int(math.Floor((pos+1)*step + 0.5))
		plan.Add(memberID, topic, partitions[min:max]...)
	}
}
```
- `BalanceStrategyRoundRobin`

```go
func(plan BalanceStrategyPlan, memberIDs []string, topic string, partitions []int32) {
	for i, part := range partitions {
		memberID := memberIDs[i%len(memberIDs)]
		plan.Add(memberID, topic, part)
	}
}
```
*不同的策略，可以实现客户端与partition的不同对应关系。*
**如果我们碰到这样一个棘手的问题:** 
需要消费同一个topic，同一个消费组，需要多个服务在不同机器上同时启动，但是机器层次不齐。当流量大时，有些机器负载比较大可能会挂机，那我们可能实现一个reblance策略，将配置高的机器多分配partition，配置低的机器少分配些partition，来满足我们如此个性化（奇葩）的需求了。

