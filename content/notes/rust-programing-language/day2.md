---
title: "Rust 编程语言 - 猜数字游戏"
date: 2023-02-01
tags:
  - rust
categories:
  - 学习笔记
type: posts
---
> 阅读[Rust程序设计语言](https://kaisery.github.io/trpl-zh-cn/title-page.html)笔记

# 输入
```rust
// 引入输入输出库到当前作用域
use std::io;

// 入口函数
fn main() {
    println!("Guess the number!");

    println!("Please input youer guess.");

    // let 创建的变量默认不可变
    // 通过增加 mut 制定变量可变
    // new 是关联函数（静态方法）
    // String::new() 申请一个可增长的字符串
    let mut guess = String::new(); 

    io::stdin()
        .read_line(&mut guess) // & 引用传递， mut 可变变量返回Result<usize, Error> （枚举类型)
        .expect("Failed to read line");

    // 占位打印
    println!("You guessed: {guess}")
}
```
`cargo run` 运行

# 生成随机数

- crate 是 rust 的库的概念；
- 检索平台 `https://crates.io/`
- `cargo build` 下载依赖
- `cargo update` 更新依赖
- `cargo doc --open` 打开文档 (这个很牛逼)


在Cargo.toml 中添加随机数的包 `rand`

```toml
[dependencies]
rand = "0.8.5"
```

程序如下：
```rust
use std::io;
use std::cmp::Ordering;

//trait
use rand::Rng;

fn main() {
    println!("Guess the number!");

    // 默认 i32 类型
    let secret_number = rand::thread_rng().gen_range(1..=100);

    println!("The secret number is: {secret_number}");

    println!("Please input youer guess.");

    let mut guess = String::new();

    io::stdin()
        .read_line(&mut guess)
        .expect("Failed to erad line");
    
    // 变量覆盖，原来的string 类型变成了 u32类型
    let guess: u32 = guess.trim().parse().expect("Please type a number");

    println!("You guessed: {guess}");

    loop {

    // match 和 分支(arms)
    match guess.cmp(&secret_number){
        Ordering::Less => println!("Too small!"),
        Ordering::Greater=> println!("Too big!"),
        Ordering::Equal=> println!("You win!"),
    }
    }
    println!("You guessed: {guess}")
}
```

# 程序控制

- 枚举类型可以通过 match, arms （分支）控制
  - match, arms 需要枚举所有可能值，否则报错
- 循环使用 loop, 通过continue 或者break 跳转循环
- Result 是包含了 Ok,Err两个枚举值的枚举类型;用处广泛
- [源代码](guessing_game/src/main.rs)

```rust
use std::io;

use rand::Rng;
use std::cmp::Ordering;

fn main() {
    println!("Guess the number!");

    // 默认 i32 类型
    let secret_number = rand::thread_rng().gen_range(1..=100);

    loop {
        println!("Please input youer guess.");

        let mut guess = String::new();
        io::stdin()
            .read_line(&mut guess)
            .expect("Failed to read line");

        println!("You guessed: {guess}");

        // 变量覆盖，原来的string 类型变成了 u32类型
        // match arms, 匹配 ok 和 err
        // parse 返回值 是Result；枚举类型，有 Ok, Err 两个成员
        let guess: u32 = match guess.trim().parse() {
            Ok(num) => num,     // 返回 数字
            Err(_) => continue, // 重新循环, _ 是通配符，匹配所有err
        };

        // match 和 分支(arms)
        match guess.cmp(&secret_number) {
            Ordering::Less => println!("Too small!"),
            Ordering::Greater => println!("Too big!"),
            Ordering::Equal => {
                println!("You win!");
                break; // 中断循环
            }
        }
        println!("You guessed: {guess}")
    }
}
```
---------
@2023-02-01
