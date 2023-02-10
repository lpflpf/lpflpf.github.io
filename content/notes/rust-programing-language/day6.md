---
title: "Rust 编程语言 - 枚举和模式匹配"
date: 2023-02-09T17:55:29+08:00
tags:
  - rust
categories:
  - 学习笔记
type: posts
lightgallery: true
keywords: 
  - 学习笔记
  - rust 结构体
---

## 枚举的定义

```rust
// 定义 ip 类型
enum IpAddrKind {
    V4,
    V6,
}

// 访问 枚举值 
let four = IpAddrKind::V4;

// 不同成员可以使用不同类型和数量的数据，可以定义为同一种类型
enum IpAddr {
        V4(u8, u8, u8, u8),
        V6(String),
}
let home = IpAddr::V4(127, 0, 0, 1);

let loopback = IpAddr::V6(String::from("::1"));

// 枚举上可以定义方法

impl IpAddr {
    fn call(&self){

    }
}
```

## Option 枚举

1. 用于判定控制的一种枚举。在标准库中定义。
2. 使用Option类型值，需要强类型转换
3. 当使用时，需要显示判断None的情况

```rust
// 可以看作是一种枚举类型，包含了None
// 在标准库中的定义形式
enum Option<T> {
    None,
    Some(T),
}

// 使用Option, 变量可为空的值
let absent_number: Option<i32> = None;
let y: Option<i8> = Some(5); 

```

## match 控制流

1. **match 必须是穷尽枚举，没有穷尽会报错**
2. `other` 对其他的处理
3. `_`忽略其他

```rust
enum Coin {
    Penny,
    Nickel,
    Dime,
    Quarter,
}

fn value_in_cents(coin: Coin) -> u8 {
    // 针对不同的场景，做不同处理
    match coin {
        Coin::Penny => {
            println!("Lucy Penny")
            1,
        }
        Coin::Nickel => 5,
        Coin::Dime => 10,
        Coin::Quarter => 25,
    }
}

// match Option
fn plus_one(x: Option<i32>) -> Option<i32> {
    match x {
        None => None,
        Some(i) => Some(i + 1),
    }
}

// 穷尽的例子

let dice_roll = 9;
match dice_roll {
    3 => add_fancy_hat(),
    7 => remove_fancy_hat(),
    other => move_player(other), // 通过other 确保穷尽
}

fn add_fancy_hat() {}
fn remove_fancy_hat() {}
fn move_player(num_spaces: u8) {}

// 忽略其他

let dice_roll = 9;
match dice_roll {
    3 => add_fancy_hat(),
    7 => remove_fancy_hat(),
    _ => reroll(),
}

fn add_fancy_hat() {}
fn remove_fancy_hat() {}
fn reroll() {}
```

## let if 简单控制流

1. 通过 let if ，减少 `_` 的样板代码；
2. 虽然代码减少了，但无法做穷尽检查;


```rust
    let config_max = Some(3u8);

    if let Some(max) = config_max {
        println!("The maximum is configured to be {}", max);
    }
```