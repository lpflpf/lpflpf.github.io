---
title: 使用原子操作保证并发安全
date: 2021-09-24 09:34:45
tags:
  - golang
---

本文介绍使用unsafe.Pointer 来提升动态加载数据的效率问题。

<!-- more -->

## 场景

在很多后台服务中，需要动态加载配置文件或者字典数据。因此在访问这些配置或者字典时，需要给这些数据添加锁，保证并发读写的安全性。正常情况下，需要使用读写锁。下面来看看读写锁的例子。

## 读写锁加载数据

使用读写锁，可以保证访问data 不会出现竞态。

``` golang
type Config struct {
        sync.RWMutex
        data map[string]interface{}
}

func (c *Config) Load() {
        c.Lock()
        defer c.Unlock()

        c.data = c.load()
}

func (c *Config) load() map[string]interface{} {
        // 数据的加载
        return make(map[string]interface{})
}

func (c *Config) Get() map[string]interface{} {
        c.RLock()
        defer c.RUnlock()
        return c.data
}
```

## 使用原子操作动态替换数据

此类业务需求有一个特点，就是读非常频繁，但是更新数据会比较少。我们可以用下面的方法替代读写锁。

```golang
import "sync/atomic"
import "unsafe"

type Config struct {
        data unsafe.Pointer
}

func (c *Config) Load() {
        v := c.load()
        atomic.StorePointer(&c.data, unsafe.Pointer(&v))
}

func (c *Config) load() map[string]interface{} {
        // 数据的加载
        return make(map[string]interface{})
}

func (c *Config) Get() map[string]interface{} {
        v := atomic.LoadPointer(&c.data)
        return *(*map[string]interface{})(v)
}
```

使用原子操作可以保证并发读写时，在更新数据时，保证新的map不会被之前的读操作获取，因此可以保证并发的安全性。

## 性能测试

下面做个性能测试，其中 ConfigV2 是使用原子操作来替换的map数据。

```golang
func BenchmarkConfig(b *testing.B) {
        config := &Config{}
        go func() {
                for range time.Tick(time.Second) {
                        config.Load()
                }
        }()

        config.Load()
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
                _ = config.Get()
        }
}

func BenchmarkConfigV2(b *testing.B) {
        config := &ConfigV2{}
        go func() {
                for range time.Tick(time.Second) {
                        config.Load()
                }
        }()

        config.Load()
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
                _ = config.Get()
        }
}
```

二者差距有40倍,结果如下:
```
goos: linux
goarch: amd64
pkg: lpflpf/loaddata
cpu: Intel(R) Xeon(R) CPU E5-2630 v3 @ 2.40GHz
BenchmarkConfig-32              551491118               21.79 ns/op            0 B/op          0 allocs/op
BenchmarkConfigV2-32            1000000000               0.5858 ns/op          0 B/op          0 allocs/op
PASS
ok      lpflpf/loaddata 14.870s
```

## 技术总结

1. 在做字典加载、配置加载的读多写少的业务中，可以使用原子操作代替读写锁来保证并发的安全。
2. 原子操作性能比较高的原因可能是: 读写锁需要多增加一次原子操作。(有待考证)
