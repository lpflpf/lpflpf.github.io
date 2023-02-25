---
title: "在go中执行 javascript 代码"
date: 2023-01-29 19:04:00
tags: 
  - golang
  - javascript
categories: 
  - golang
---

**filebeat的javascript插件是怎么run起来的?**   
filebeat 是由golang 实现的，其间接引用了`https://github.com/dop251/goja`包，该包使用纯golang语言实现了 ECMAScript 5.1.  

内嵌脚本语言，其实并不少见。比如常用的Lua，就被内嵌在redis中。javascript 相对于lua，是非常灵活的，学习成本也相对较低。

经过调研，实现javascript运行时的包有如下几个：

- `github.com/robertkrimen/otto` [Star 7.1K]
  - 最初使的项目
  - 性能较低
  - go 1.18
  - ECMA 不支持严格模式
  - 正则不完全兼容
  - es6 特性不支持
- `github.com/dop251/goja` [Star 3.5K]
  - filebeat使用
  - 思想来源于otto
  - go 1.16
  - ECMAScript 5.1 引擎
  - 支持 regex and strict mode
  - 在频繁执行较简单的js情况下，性能与otto基本持平，高于v8go
- `github.com/rogchap/v8go` [Star 2.5K]
  - 基于google的V8引擎实现(chrome)
  - cgo实现，性能最高(在需要执行较长的js代码的情况下),与系统兼容性略差.（windows可能需要自己编译v8)
  - javascript兼容性比较好
  - 调试困难

### otto case

```golang
package main

import (
        "fmt"

        "github.com/robertkrimen/otto"
)

func main() {
        vm := otto.New()

        // 注入变量
        vm.Set("def", map[string]interface{}{"abc": 123})
        // 注入方法
        vm.Set("Add", func(call otto.FunctionCall) otto.Value {
                var a, b int64
                a, _ = call.Argument(0).ToInteger()
                b, _ = call.Argument(1).ToInteger()

                val, _ := vm.ToValue(a + b)
                return val
        })

        // 执行
        vm.Run(`
                abc = Add(1,2);
                console.log("The value of abc is " + abc);
                console.log("The value of def is " , def.abc);
        `)

        // 变量取值
        if value, err := vm.Get("abc"); err == nil {
                if intVal, err := value.ToInteger(); err == nil {
                        fmt.Println(intVal)
                }
        }
}
```

### goja case

```golang
package main

import (
        "fmt"

        "github.com/dop251/goja"
)

func main() {
        vm := goja.New()
        // 返回值
        v, _ := vm.RunString("2+2")
        fmt.Println(v.Export().(int64))

        // 注入方法
        vm.Set("add", func(call goja.FunctionCall) goja.Value {
                var a, b int64
                a = call.Argument(0).ToInteger()
                b = call.Argument(1).ToInteger()

                val := vm.ToValue(a + b)
                return val
        })

        v, _ = vm.RunString(`add(1,2)`)
        fmt.Println(v.Export().(int64))

        // 导出方法
        vm.RunString(`
function sub(a,b) {
        return a - b
}
        `)

        sub, _ := goja.AssertFunction(vm.Get("sub"))

        v, _ = sub(goja.Undefined(), vm.ToValue(10), vm.ToValue(1))

        fmt.Println(v.Export().(int64))
        
}
```

### v8go

```golang

package main

import (
        "fmt"

        v8 "rogchap.com/v8go"
)

func main() {
        iso := v8.NewIsolate()
        global := v8.NewObjectTemplate(iso)

        // 注入方法
        fn := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
                for _, v := range info.Args() {
                        fmt.Println(v)
                }
                val, _ := v8.NewValue(iso, "something")
                return val
        })

        // 注入变量
        abc, _ := v8.NewValue(iso, int32(456))

        global.Set("abc", abc, v8.ReadOnly)
        global.Set("print", fn, v8.ReadOnly)
        ctx := v8.NewContext(iso, global)
        defer ctx.Close()
        ctx.RunScript(`
print("abc", 123, {"abc":123});
print(abc)
        `, "")

        //获取返回结果

        resp, err := ctx.RunScript(`abc + 321`, "")
        fmt.Println(resp.Int32(), err)
}
```

## 性能测试

主要两个方面做性能测试：
1. 测试简单的加法操作，主要测试高频的启动js引擎的操作
2. 测试js的高频计算，测试不同js内部的执行效率

[压测源码](https://github.com/lpflpf/lpflpf.github.io/tree/main/content/posts/javascript-run-in-go/benchmark)

对比如下：

```
goos: darwin
goarch: arm64
pkg: javascripttest
BenchmarkOttoAdd-10        19362             61799 ns/op
BenchmarkGojaAdd-10        15794             75839 ns/op
BenchmarkV8Add-10           4783            260458 ns/op
BenchmarkSumOtto-10           15          70687100 ns/op
BenchmarkSumGoja-10           67          17269518 ns/op
BenchmarkSumV8-10            489           2203052 ns/op
PASS
ok      javascripttest  9.329s
```
