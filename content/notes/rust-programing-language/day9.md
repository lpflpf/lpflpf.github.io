---
draft: true
title: "Rust 编程语言 - 错误处理"
date: 2023-02-17
tags:
  - rust
categories:
  - 学习笔记
type: posts
lightgallery: true
keywords: 
  - 学习笔记
  - 错误处理
---

两种类型的错误，可恢复的和不可恢复的。

可恢复的用`Result<T,E>`处理,不可恢复的使用`panic!`宏处理

## 处理不可恢复的错误

- panic 时，默认会回溯调用栈，并打印调用展；若为了将bin文件变小，可以在配置中设置`panic = 'abort'`, 直接终止
- panic 可以手动调用，也可能是异常触发的(例如数组访问超索引)
- 一般会保留调用栈，方便问题排查
- 通过 RUST_BACKTRACE 环境变量打印 调用栈（必须是debug 模式）

## 处理可恢复的错误

Result 处理可恢复的错误, Result 是枚举类型
```rust
enum Result<T, E> {
    Ok(T),
    Err(E),
}
```

例如，使用match 对Result的处理

```rust
use std::fs::File;
use std::io::ErrorKind;

fn main() {
    let greeting_file_result = File::open("hello.txt");

    let greeting_file = match greeting_file_result {
        Ok(file) => file,
        // 枚举类型 ErrorKind
        Err(error) => match error.kind() {
            ErrorKind::NotFound => match File::create("hello.txt") {
                Ok(fc) => fc,
                Err(e) => panic!("Problem creating the file: {:?}", e),
            },
            other_error => {
                panic!("Problem opening the file: {:?}", other_error);
            }
        },
    };

    // 采用闭包的方式处理, 代码会少很多
    let greeting_file = File::open("hello.txt").unwrap_or_else(|error| {
        if error.kind() == ErrorKind::NotFound {
            File::create("hello.txt").unwrap_or_else(|error| {
                panic!("Problem creating the file: {:?}", error);
            })
        } else {
            panic!("Problem opening the file: {:?}", error);
        }
    });
}
```
### 失败时panic,而不处理

Result 实现的一些方法。

- `unwarp()` 若失败，则panic
- `expect("")` 可以增加报错提示信息

### 错误传播

```rust
use std::fs::File;
use std::io::{self, Read};

fn read_username_from_file() -> Result<String, io::Error> {
    let username_file_result = File::open("hello.txt");

    let mut username_file = match username_file_result {
        Ok(file) => file,
        // 失败，直接返回
        Err(e) => return Err(e),
    };

    let mut username = String::new();
    match username_file.read_to_string(&mut username) {
        Ok(_) => Ok(username),
        Err(e) => Err(e),
    }
}

// 简化版本
use std::fs::File;
use std::io::{self, Read};

fn read_username_from_file() -> Result<String, io::Error> {
    let mut username_file = File::open("hello.txt")?; // 如果是Err 则返回
    let mut username = String::new();
    username_file.read_to_string(&mut username)?; // 问号符号
    Ok(username)
}

// 链式之后直接调用
use std::fs::File;
use std::io::{self, Read};

fn read_username_from_file() -> Result<String, io::Error> {
    let mut username = String::new();

    File::open("hello.txt")?.read_to_string(&mut username)?;

    Ok(username)
}
```

**必须在返回值为Result 类型的方法中使用?链式调用**

## 是否需要panic

1. 示例、代码原型和测试都非常适合 panic
2. 有害状态下适合使用panic，有害状态指：
   1. 当一些假设、保证、协议或不可变性被打破的状态，比如无效值，自相矛盾的值，不存在的值等；
   2. 有害状态是非预期的行为，与偶尔会发生的行为相对，比如用户输入了错误格式的数据
   3. 后续代码依赖的值非预期行为
   4. 没有可行的手段来将有害状态信息编码进所使用的类型中的情况
3. 当能预期到错误出现时，返回Result更合适