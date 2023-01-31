---
title: "Rust 编程语言"
date: 2023-01-31T12:38:22+08:00
tags:
  - rust
category:
  - 学习笔记
---

> 阅读[Rust程序设计语言](https://kaisery.github.io/trpl-zh-cn/title-page.html)笔记


## 入门指南
### 安装

- 下载rustup脚本，并安装
`curl --proto '=https' --tlsv1.2 https://sh.rustup.rs -sSf | sh` 
- 或者直接`brew install rust`安装
- 通过 `rustc --verison` 查看是否成功

```s
rustc 1.66.0 (69f9c33d7 2022-12-12) (built from a source tarball)
```

### hello world

```rust
// file main.rs

// main 是函数入口
fn main() {

    // println! 有感叹号是 rust 的宏
    // ; 语句结尾
    println!("Hello, world!");
}
```

执行 `rustc main.rs` 编译，`./main`执行; 可以通过`rustfmt` 格式化代码

### Cargo 

rust 的构建系统和包管理器。


- cargo 创建项目 `cargo new hello`; 会生成 Cargo.toml 以及一个`src/main.rs`
```
# cat Cargo.toml
[package]
name = "hello" # 名称
version = "0.1.0" # 版本
edition = "2021" # rust 版本, 基于2021 的版本，并向后兼容

# See more keys and their definitions at https://doc.rust-lang.org/cargo/reference/manifest.html

[dependencies]
```

- 编译 `cargo build`; 二次编译会缓存,不需要重新编译
- 直接运行 `cargo run` 
- 静态检查 `cargo check`; 检查是否可编译通过
- 发布 `cargo build --release`; 会做编译优化


----
*@2023-01-31*


## 猜数字游戏

```rust
// 引入输入输出库到当前作用域
use std::io;

fn main() {
    println!("Guess the number!");

    println!("Please input youer guess.");

    // let 创建的变量默认不可变
    // 通过增加 mut 制定变量可变
    // new 是静态方法
    // String::new() 申请一个可增长的字符串
    let mut guess = String::new(); 

    io::stdio()
        .read_line(&mut guess) // & 引用传递， mut 可变变量
        .expect("Failed to read line");

    println!("You guessed: {guess}")
}
```