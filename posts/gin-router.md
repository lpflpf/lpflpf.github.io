---
title: Go Web 框架 Gin 路由的学习
date: 2020-06-05 17:42:56
tags:
    - gin
    - golang
    - http router
---


本文主要从源码角度介绍 Gin 框架路由的实现。

<!-- more -->
Gin 是目前应用比较广泛的Golang web 框架。 目前，Github Star 数已经达到了3.8w. 框架的实现非常简单，可定制性非常强，性能也比较好，深受golang开发者的喜爱。Gin 提供了web开发的一些基本功能。如路由，中间件，日志，参数获取等，本文主要从源码的角度分析Gin的路由实现。

Gin 的路由功能是基于 `https://github.com/julienschmidt/httprouter` 这个项目实现的。目前也有很多其他Web框架也基于该路由框架做了二次开发。

# http 路由的接口

在 Gin 中，为了兼容不同路由的引擎，定义了 IRoutes 和 IRouter 接口，便于替换其他的路由实现。（目前默认是httprouter)

下面是一个路由的接口定义

```go
type IRoutes interface {
   Use(...HandlerFunc) IRoutes

  Handle(string, string, ...HandlerFunc) IRoutes
  Any(string, ...HandlerFunc) IRoutes
  GET(string, ...HandlerFunc) IRoutes
  POST(string, ...HandlerFunc) IRoutes
  DELETE(string, ...HandlerFunc) IRoutes
  PATCH(string, ...HandlerFunc) IRoutes
  PUT(string, ...HandlerFunc) IRoutes
  OPTIONS(string, ...HandlerFunc) IRoutes
  HEAD(string, ...HandlerFunc) IRoutes

  StaticFile(string, string) IRoutes
  Static(string, string) IRoutes
  StaticFS(string, http.FileSystem) IRoutes
}

type HandlerFunc func(*Context)
```

HandlerFunc 是一个方法类型的定义，我们定义的路由其实就是一个路径与HandlerFunc 的映射关系。
从上面的定义可以看出，IRoutes 主要定义了一些基于http方法、静态方法的路径和一组方法的映射。 `Use` 方法是针对此路由的所有路径映射一组方法，在使用上是为了给这些路由添加中间件。

除了上面的定义外，Gin 还有路由组的抽象。

```go
type IRouter interface {
  IRoutes
  Group(string, ...HandlerFunc) *RouterGroup
}
```

路由组是在IRoutes 的基础上，有了组的概念，组下面还可以挂在不同的组。组的概念可以很好的管理一组路由，路由组可以自己定义一套Handler方法（即一组中间件）。

*个人认为IRouter的定义Group 应该返回 IRouter，这样可以把路由组更加抽象，也不会改变现有服务的使用。期待看下Gin源码什么时候会按照这种定义方法修改过来。*

在Gin框架中，路由由 RouterGroup 实现。我们从构造和路由查找两个方面分析路由的实现。

# 路由实现

路由的本质就是在给定 路径与Handler映射关系 的前提下，当提供新的url时，给出对应func 的过程。其中可能需要从url中提取参数，或者按照 `*` 匹配 url 的情况。

首先，我们看下Gin中路由结构的定义。

```go
// gin engine
type Engine struct {
  RouterGroup
  // ... 其他字段
  trees            methodTrees
}

// 每个 http 方法定义一个森林
type methodTrees []methodTree

type methodTree struct {
  method string
  root   *node
}

// 路由组的定义
type RouterGroup struct {
  Handlers HandlersChain
  basePath string
  engine   *Engine
  root     bool
}

```

从定义中可以看出，其实Gin 的 Engine 是复用了 RouterGroup。对于不同的 http method，都通过一个森林来存储路由数据。
下面是森林上每个节点的定义：

```go
type node struct {
  path      string  // 当前路径
  indices   string  // 对应children 的前缀
  wildChild bool   // 可能是带参数的，或者是 * 的，所以是野节点
  nType     nodeType  // 参数节点，静态节点
  priority  uint32  // 优先级 ，优先级高的放在children 放在前面。
  children  []*node  // 子节点
  handlers  HandlersChain // 调用链
  fullPath  string  // 全路径
}
```

从代码实现上得知，这个森林其实是一个压缩版本的Trie树，每个节点会存储前缀相同的路径数据。下面，我们通过代码来学习下路由的添加和删除。

## 路由的添加

路由的添加，就是将path路径添加到定义的Trie树种，将handlers 添加到对应的node 节点。

```go
func (n *node) addRoute(path string, handlers HandlersChain) {
  // 初始化和维护优先级

  for {
    // 查找前缀
    i := longestCommonPrefix(path, n.path)

 // 原有路径长的情况下
 // 节点n 的 path 变为了公共前缀
 // 原有n 的path 路径变为了现有n 的子节点

 // 当添加的path长的情况
 // 需要分情况讨论：
 // 1. 如果是一个带参数的路径，校验是否后续路径不同，如果不同则继续扫描下一段路径
 // 2. 如果是带 * 的路径， 则直接报错
 // 3. 如果已经有对应的首字母，修改当前node节点，并继续扫描，并扫描下一段路径
 // 4. 如果非参数或者 * 匹配的方法，则插入一个子节点路径，并完成扫描

 // 最后注册handlers，添加fullPath
    n.handlers = handlers
    n.fullPath = fullPath
    return
  }
}
```

从上面的代码注释可以看出，路由的添加，主要是通过不断对比当前节点的path和添加的path，做添加节点或者节点变更的操作，达到添加path的目的。

## 路径查找

在服务请求时，路由的责任就是给定一个url请求，拿到节点保存的handlers，以及url中包含的参数值。下面是对一个url 的解析实现。

```go
type nodeValue struct {
	handlers HandlersChain
	params   *Params
	tsr      bool
	fullPath string
}

func (n *node) getValue(path string, params *Params, unescape bool) (value nodeValue) {
walk: // Outer loop for walking the tree
  for {
    prefix := n.path

// 如果比当前节点路径要长：
//  - 非参数类型或模糊匹配的URL，如果和当前节点前缀匹配，直接查看 node 的子节点
//  - 参数化的node, 按照 / 分割提取参数，如果未结束，则继续匹配剩下的路径，否则返回结果。
//  - * 匹配的node，将剩余的路径添加到 param 中直接返回。
// 如果和当前节点相等，那就直接返回即可。
// 这里还做了非本方法的路径匹配，用户返回http 方法错误的异常报告。
  }
}
```

## 一个例子

下面通过一个例子，方便我们快速理解router的实现。

加入下面的一个路径：
  /search/
  /support/
  /blog/:post/
  /about-us/team/
  /contact/

在树中,我们看到的样子如下：

```go
  Path
  \
  ├s
  |├earch\
  |└upport\
  ├blog\
  |    └:post
  |         └\
  ├about-us\
  |        └team\
  └contact\
```

在做路由查找时，通过路径不断匹配，找到对应的子节点。拿到对应子节点下的handler。完成路由的匹配。

# 总结

1. httprouter 没有实现了routergroup功能，只是实现了router 的功能，在gin中做了实现
2. 通过Trie树实现路由是比较基础的一种实现方法，除了这种方法外，还可以考虑通过正则的方式提取路由。
3. Gin http 服务是基于 Go 的 `net/http` 库的， `net/http` 库中handler 的实现是针对不同的 http method 的，所以需要在engine 中针对不同的method 提供不同的trie 树。
4. 在添加路由时，如果使用了 any 方法，则在每个http method 下都会添加一样的路径。
5. middleware 本质上只是一个 HandlerFunc.

