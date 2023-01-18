---
 layout: post
 title: "Go Base Syntax"
 subtitle: "basic structure"
 date: 2018-06-15
 author: "李朋飞"
 tags:
   - golang
   - base structure

 category: golang
---


### go程序

go程序说明

```go
    package main    
    // 程序包名, 与 php namespace 类似； 和java 相同    
      
    // import 可以通过 { } 导入多个包。 中间加入 ".", 可以在引用函数时，不带包名  
    import . "fmt"  
    // 引入包名重命名 (.) 可以认为是类似的引用.  
    import myio  "io"  
      
    // 定义常量  
    const PI = 3.14  
      
    // 定义一般变量  
    var name = "gopher"  
      
    // 申明类型newType 为 int； 类似于C 中typedef  
    type newType int  
      
    // 申明类型gopher 为 一个空结构体  
    type gopher struct{}  
      
    // 申明golang接口  
    type golang interface{}  
      
    // 函数以func 开头，类似于PHP 中function  
    func main() {  
        Println ("Hello World")  
    }  
      
    // 大写开头的函数，可以在包外引用  
    func SayHello(){  
      
    }  
```
