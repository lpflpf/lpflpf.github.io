---
title: "Rust 编程语言 - 认识所有权"
date: 2023-02-03
tags:
  - rust
categories:
  - 学习笔记
type: posts
lightgallery: true
keywords: 
  - rust 所有权
  - 学习笔记
---

> 阅读[Rust程序设计语言](https://kaisery.github.io/trpl-zh-cn/title-page.html)笔记

## 什么是所有权

**所有权(ownership)的好处**：可以不使用垃圾回收(garbage collector)，即可保障内存安全。

{{< admonition type=tip title="规则" open=true >}}
> - Rust 每个值都有一个所有者
> - 值在任何时刻都有且仅有一个所有者
> - 所有者离开作用域，值将被丢弃
{{< /admonition >}}

### 变量与数据的交互方式 - 移动
- drop 函数是释放内存的函数，在变量离开作用域后，自动调用释放内存；
- 对于整形的变量拷贝，由于不可变，会放入栈中
- 对于string的拷贝复制,则只是拷贝了头指针
  - 字符串赋值后，之前的变量不再有效，防止double drop
```rust
  let s1 = String::from("hello");
  let s2 = s1; // 指针移动(move)
```
{{<figure src="day4-1.png" width="400" >}}

### 变量与数据的交互方式 - 克隆

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
### 函数与所有权

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


## 引用与借用

1. (默认)引用不许修改引用的值
```rust
fn main() {
    let s1 = String::from("hello");
    let len = calculate_length(&s1); 
    println!("The length of '{}' is {}.", s1, len);
}

// s 是引用
fn calculate_length(s: &String) -> usize {
    s.len()
}
```
{{<figure src="day4-2.png" width="600" >}}

### 可变引用

1. 通过增加 mut 修饰，使引用的值可以被修改；
2. 如果已经有一个对变量的可变引用，同一个作用域不能再增加对该变量的引用；（避免数据竞争）
3. 不同作用域可以多次采用引用

### 悬垂引用

```rust
fn main() {
    let reference_to_nothing = dangle();
}

fn dangle() -> &String {
    let s = String::from("hello");

    &s  // 返回了引用，变量变成了悬垂状态，引用变量无效，会编译器报错
        // 修改成返回 String 类型
}
```

### 总结

1. 在任意给定时间，要么 只能有一个可变引用，要么 只能有多个不可变引用。
2. 引用必须总是有效的。

## Slice 类型

```rust
    let s = String::from("hello world");

    let hello = &s[0..5];
    let world = &s[6..11];
```