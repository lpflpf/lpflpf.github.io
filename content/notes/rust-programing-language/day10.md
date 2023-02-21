---
title: "Rust 编程语言 - 范型、Trait和生命周期"
date: 2023-02-21
tags:
  - rust
categories:
  - 学习笔记
type: posts
lightgallery: true
description: "rust 范型、Trait和生命周期"
keywords: 
  - 学习笔记
  - 范型
  - 生命周期
  - Trait
---

## 范型数据类型

1. 一般使用T作为类型参数名称
2. 适用于结构体、枚举、函数等
3. 支持多个类型
4. 可以限定方法实现

```rust
fn largest<T>(list: &[T]) -> &T {
    let mut largest = &list[0];

    for item in list {
        if item > largest {
            largest = item;
        }
    }

    largest
}

struct Point<T,U> {
    x: T,
    y: U,
}
```

Rust 通过单态化保证运行效率。将通用代码转化为特定代码。

## Trait 定义共同行为


1. 类似于高级语言中的接口
2. 可以指定默认实现
3. 不同的类型可以冲在该方法


```rust
pub trait Summary {
    fn summarize(&self) -> String;
}


// 包含默认实现
pub trait Summary {
    fn summarize(&self) -> String {
        String::from("(Read more...)")
    }
}

// 范型中，实现了Summary Trait 的类型T
pub fn notify<T: Summary>(item: &T) {
    println!("Breaking news! {}", item.summarize());
}

// 实现了 Summary + Display 类型的参数
pub fn notify(item: &(impl Summary + Display)) {}


// where 语句，指定Trait
fn some_function<T, U>(t: &T, u: &U) -> i32
where
    T: Display + Clone,
    U: Clone + Debug,
{}
```

## 声明周期确保引用有效

**目的**：确保悬垂引用

*悬垂引用*
```rust
fn main() {
    let r;

    {
        let x = 5;
        // r 引用了x,x 尝试离开作用域,
        r = &x;
    }

    println!("r: {}", r);
}
```

**生命周期注解**
```rust
&i32        // 引用
&'a i32     // 带有显式生命周期的引用
&'a mut i32 // 带有显式生命周期的可变引用

struct ImportantExcerpt<'a> {
    part: &'a str,
}

fn longest<'a>(x: &'a str, y: &str) -> &'a str {
    x
}
```

编译器采用三条规则来判断引用何时不需要明确的注解:
1. 编译器为每一个是引用参数都分配了一个生命周期参数
2. 如果只有一个输入生命周期参数，那么它被赋予所有输出生命周期参数：`fn foo<'a>(x: &'a i32) -> &'a i32`。
3. 如果方法有多个输入生命周期参数并且其中一个参数是 &self 或 &mut self，说明是个对象的方法 (method)(译者注：这里涉及 rust 的面向对象参见 17 章)，那么所有输出生命周期参数被赋予 self 的生命周期;