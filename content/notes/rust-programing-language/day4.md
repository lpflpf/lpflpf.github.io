---
title: "认识所有权"
date: 2023-02-03
tags:
  - rust
categories:
      - 学习笔记
---

> 阅读[Rust程序设计语言](https://kaisery.github.io/trpl-zh-cn/title-page.html)笔记

# 什么是所有权

**所有权(ownership)的好处**：可以不使用垃圾回收(garbage collector)，即可保障内存安全。

> 规则：
> - Rust 每个值都有一个所有者
> - 值在任何时刻都有且仅有一个所有者
> - 所有者离开作用域，值将被丢弃

## 变量与数据的交互方式 - 移动
- drop 函数是释放内存的函数，在变量离开作用域后，自动调用释放内存；
- 对于整形的变量拷贝，由于不可变，会放入栈中
- 对于string的拷贝复制,则只是拷贝了头指针
  - 字符串赋值后，之前的变量不再有效，防止double drop
```rust
  let s1 = String::from("hello");
  let s2 = s1; // 指针移动(move)
```
![](day4-1.png)

## 变量与数据的交互方式 - 克隆

克隆，数据深度拷贝
- 堆上数据的拷贝，需要通过clone 深度拷贝
- 栈上的数据，可以直接复制
- Copy Trait （实现该trait 可以保证直接复制，而不会导致原油变量失效）
- 如果实现了Drop Trait，则copy Trait 不能再使用

```rust
    let s1 = String::from("hello");
    let s2 = s1.clone();

    println!("s1 = {}, s2 = {}", s1, s2);
```
## 函数与所有权

```rust
fn main() {
    let s = String::from("hello");  // s 进入作用域

    takes_ownership(s);             // s 的值移动到函数里 ...
                                    // ... 所以到这里不再有效

    let x = 5;                      // x 进入作用域

    makes_copy(x);                  // x 应该移动函数里，
                                    // 但 i32 是 Copy 的，
                                    // 所以在后面可继续使用 x

} // 这里，x 先移出了作用域，然后是 s。但因为 s 的值已被移走，
  // 没有特殊之处

fn takes_ownership(some_string: String) { // some_string 进入作用域
    println!("{}", some_string);
} // 这里，some_string 移出作用域并调用 `drop` 方法。
  // 占用的内存被释放

fn makes_copy(some_integer: i32) { // some_integer 进入作用域
    println!("{}", some_integer);
} // 这里，some_integer 移出作用域。没有特殊之处
```

当 s 被传入函数后，函数调用后不可以再使用s变量；
函数返回值也是如此，会出现所有权的转移


# 引用与借用