---
title: Golang反射学习：手写一个RPC
date: 2021-01-08 16:03:00
tags:
  - golang
  - reflect

category: golang
---

本文主要为了在对golang反射学习后做一个小练习，使用100行代码实现一个通用的RPC服务。

<!--more-->

## 简要说明

golang 的RPC框架还是非常丰富的，比如 gRPC，go-zero, go-dubbo 等都是使用非常普遍的rpc框架。在go语言实现的RPC客户端中，大部分RPC框架采用的是使用生成代码的方式来构建RPC服务。即：定义好相应的接口后，需要通过命令生成相应的代码。采用这种方式的优点在于可以减少不必要的类型转换；而麻烦之处也显而易见，需要在每次结构发生改变时，重新生成对应的代码。那么，如果不采用命令行生成的方式来调用RPC该怎么做呢？经过对golang反射的学习后，让我们用100行代码来小试牛刀，实现一个极简版的RPC。

## 协议的定义
由于极简，我们采用HTTP协议，数据传输采用最常见的json结构。
服务请求，通过http 请求路径判断调用哪个方法。

- 输入参数定义如下：
`["参数1", "参数2"]`
其中参数1，参数2 采用Json编码，最终的请求参数在做一次编码。
例如：`Do("abc", 123)` ,其post请求body为 `["\"abc\"", 123]`

- 输出参数定义与输入参数定义格式相同


## 如何使用

### 服务端：

服务端需要实现每一个接口，并把接口绑定到对应的路由上。

```go
package main

import "github.com/lpflpf/rpc"
import "strconv"

func main() {
  serv := rpc.NewRpcServ("127.0.0.1:18080")
  serv.Impl("/conv/int2str", strconv.Itoa) // 路由绑定到方法
  serv.Impl("/conv/str2int", strconv.Atoi)
  serv.Impl("/math/add", func(a, b int) int { return a + b })
  serv.Start()
}
```

### 客户端调用

客户端仅需要定义对应的rpc服务的方法，并通过struct tag的方式指定路由即可

```go
package main

import "fmt"
import "github.com/lpflpf/rpc"

type Conv struct {
  Int2Str func(int) string                `rpc:"conv/int2str"`
  Str2Int func(input string) (int, error) `rpc:"conv/str2int"`
}

type Math struct {
  Add func(int, int) int `rpc:"math/add"`
}

func main() {
  conv := &Conv{}
  rpc.Connect("http://127.0.0.1:18080", conv) // 连接RPC 服务
  fmt.Println(conv.Int2Str(123), conv.Int2Str(456)) // 123 456
  fmt.Println(conv.Str2Int("1234")) // 1234 <nil>

  math := &Math{}
  rpc.Connect("http://127.0.0.1:18080", math) // 连接 RPC 服务
  fmt.Println(math.Add(1, 2)) // 3
}
```


## Server 端的实现

服务端主要是将注册路由。在处理请求时，需要将请求的数据转化为注册句柄的参数，并将句柄的处理结果编码，并返回给客户端。代码如下：


```go
package rpc

import "net/http"
import "reflect"
import "encoding/json"
import "io/ioutil"

type RpcServ struct {
  serv *http.Server
  mux  *http.ServeMux
}

func (rs *RpcServ) Impl(router string, f interface{}) {
  rs.mux.HandleFunc(router, func(rw http.ResponseWriter, request *http.Request) {
    rt := reflect.TypeOf(f)
    requestBody, _ := ioutil.ReadAll(request.Body)

    requestData := []string{}
    _ = json.Unmarshal(requestBody, &requestData)

    params := []reflect.Value{}

    num := rt.NumIn()
    if rt.IsVariadic() {
      num = num - 1
    }
    for i := 0; i < num; i++ {
      val := reflect.New(rt.In(i))
      json.Unmarshal([]byte(requestData[i]), val.Interface())
      params = append(params, val.Elem())
    }

    call := reflect.ValueOf(f)
    result := []reflect.Value{}

    if rt.IsVariadic() {
      val := reflect.MakeSlice(rt.In(num), 0, 0).Interface()
      json.Unmarshal([]byte(requestData[num]), &val)
      params = append(params, reflect.ValueOf(val))
      result = call.CallSlice(params)
    } else {
      result = call.Call(params)
    }

    response := []string{}
    for _, res := range result {
      val, _ := json.Marshal(res.Interface())
      response = append(response, string(val))
    }

    data, _ := json.Marshal(response)
    rw.Write(data)
  })
}

func (rs *RpcServ) Start() {
  rs.serv.Handler = rs.mux
  rs.serv.ListenAndServe()
}

func NewRpcServ(addr string) *RpcServ {
  return &RpcServ{
    serv: &http.Server{Addr: addr},
    mux:  http.NewServeMux(),
  }
}

```

## 客户端实现

客户端需要在Connect时，针对定义的每个句柄（即客户端调用时内部的方法）均需要绑定一个RPC 请求的实现。

RPC 请求的实现，即获取方法调用的各个参数，并编码后发送请求至 server 端，读取请求结果并解码，将解码后的数据填充为函数的返回值。

```go
package rpc

import "io/ioutil"

import "bytes"
import "strings"
import "reflect"
import "errors"
import "encoding/json"
import "net/http"

type RpcClient struct {
  serv *http.Server
  mux  *http.ServeMux
}

// struct BIND RPC
func Connect(addr string, iface interface{}) error {
  rv := reflect.ValueOf(iface).Elem()
  rt := reflect.TypeOf(iface).Elem()

  if rt.Kind() != reflect.Struct {
    return errors.New("")
  }

  for i := 0; i < rt.NumField(); i++ {
    if requestPath := rt.Field(i).Tag.Get("rpc"); requestPath == "" {
      continue
    } else {
      fieldType := rt.Field(i).Type
      rv.Field(i).Set(reflect.MakeFunc(fieldType, func(params []reflect.Value) []reflect.Value {
        requestBody := []string{}
        for _, param := range params {
          raw, _ := json.Marshal(param.Interface())
          requestBody = append(requestBody, string(raw))
        }
        body, _ := json.Marshal(requestBody)

        // 拼接请求Uri
        requestUri := strings.Trim(addr, "/") + "/" + strings.Trim(requestPath, "/")
        resp, _ := http.Post(requestUri, "application/json", bytes.NewReader(body))
        defer resp.Body.Close()
        data, _ := ioutil.ReadAll(resp.Body)

        // 组装返回结果
        ret := []reflect.Value{}
        responseStr := []string{}
        _ = json.Unmarshal(data, &responseStr)
        for i := 0; i < fieldType.NumOut(); i++ {
          val := reflect.New(fieldType.Out(i))
          _ = json.Unmarshal([]byte(responseStr[i]), val.Interface())
          ret = append(ret, val.Elem())
        }

        return ret
      }))
    }
  }

  return nil
}
```

## 小记

在看reflect.MakeFunc 时，源码中给出的例子是一个抽象的Swap 方法，联想到可以通过抽象的方法来实现一个RPC的调用。因此有了本文中的代码。
代码中未做异常处理，仅是对reflect.MakeFunc, reflect.Call, reflect.CallSlice 理解的一个实践。

-----
- **golang 版本**: go1.12.5 linux/amd64
- **源码地址**: [github.com/lpflpf/rpc](//github.com/lpflpf/rpc)
