---
title: "Rust 编程语言 - 智能指针"
date: 2023-03-02
tags:
  - rust
categories:
  - 学习笔记
type: posts
lightgallery: true
description: "智能指针"
keywords: 
  - 学习笔记
  - 智能指针
---
> 智能指针起源于C++，并存在于其他语言。Rust 定义了多种不同的智能指针，并提供了多于引用的额外功能。  
> 普通引用和智能指针：引用是一类只借用数据的指针；智能指针拥有他们指向的数据



## 使用 Box<T>指向堆上的数据 

使用场景：
1. 当有一个在编译时未知大小的类型，而又想要在需要确切大小的上下文中使用这个类型值的时候
2. 当有大量数据并希望在确保数据不被拷贝的情况下转移所有权的时候
3. 当希望拥有一个值并只关心它的类型是否实现了特定 trait 而不是其具体类型的时候

**可存储递归数据**
```rust
enum List {
    // 若不使用Box 则会犹豫无法计算内存大小而报错
    // 通过使用Box 可以计算出，Cons 成员需要 Box指针 + 一个int32 大小的空间
    Cons(i32, Box<List>),
    Nil,
}

use crate::List::{Cons, Nil};

fn main() {
    let list = Cons(1, Box::new(Cons(2, Box::new(Cons(3, Box::new(Nil))))));
}
```

## 使用Deref trait 将智能指针当作常规引用处理

常规引用,类似于指针，可以通过解引用访问数据:
```rust
let x = 5;
let y = &x;
assert_eq!(5, *y); // 解引用
```
Box 也可以使用解引用访问数据:
```rust
let x = 5;
let y = Box::new(x);

assert_eq!(5, x);
assert_eq!(5, *y);
```

### 自定义智能指针

类似于Box 的new方法
```rust
struct MyBox<T>(T);

impl<T> MyBox<T> {
    fn new(x: T) -> MyBox<T> {
        MyBox(x)
    }
}
```

## 相关文章

- [Rust 程序设计语言](https://kaisery.github.io/trpl-zh-cn/ch15-00-smart-pointers.html)
