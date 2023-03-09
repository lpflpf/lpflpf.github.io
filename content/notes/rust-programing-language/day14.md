---
title: "Rust 编程语言 - 无畏并发"
date: 2023-03-09
tags:
  - rust
categories:
  - 学习笔记
type: posts
lightgallery: true
description: "rust 并发学习"
keywords: 
  - 学习笔记
  - 无畏并发
---

## 使用线程同时运行代码

多线程会遇到的问题：
- 竞态条件
- 死锁
- 只在特定条件下发生，难以复现和修复的bug

**rust 标准库使用1:1实现线程。一个语言级别的线程对应一个操作系统线程**

### 创建线程

使用 `thread::spawn`创建线程
```rust
use std::thread;
use std::time::Duration;

fn main() {
    // 闭包方法创建线程
    thread::spawn(|| {
        for i in 1..10 {
            println!("hi number {} from the spawned thread!", i);
            thread::sleep(Duration::from_millis(1));
        }
    });

    for i in 1..5 {
        println!("hi number {} from the main thread!", i);
        thread::sleep(Duration::from_millis(1));
    }
}
```

### 等待线程终止

使用 `join` 方法等待线程终止

```rust
use std::thread;
use std::time::Duration;

fn main() {
    let handle = thread::spawn(|| {
        for i in 1..10 {
            println!("hi number {} from the spawned thread!", i);
            thread::sleep(Duration::from_millis(1));
        }
    });

// 等待线程结束后执行main方法 
    handle.join().unwrap();
    for i in 1..5 {
        println!("hi number {} from the main thread!", i);
        thread::sleep(Duration::from_millis(1));
    }

}
```

### 传递参数

使用move 关键字，强制将依赖的参数的所有权交给线程内部
```rust
use std::thread;

fn main() {
    let v = vec![1, 2, 3];

    let handle = thread::spawn(move || {
        println!("Here's a vector: {:?}", v);
    });

    handle.join().unwrap();
}
```

## 使用消息传递

go 思想:**不要通过共享内存来通讯；而是通过通讯来共享内存。**

```rust
use std::sync::mpsc;
use std::thread;

// mpsc 产出可以支持多个生产者，一个消费者
fn main() {
    let (tx, rx) = mpsc::channel();

    // rx 可以通过clone 实现多个生产者
    thread::spawn(move || {
        let val = String::from("hi");
        // tx 是生产端
        tx.send(val).unwrap();

        // 之后val 不可再被使用，所有权被移动到接收者
    });

    // 接收端 阻塞接收
    // rx.try_recv 非阻塞接收
    let received = rx.recv().unwrap();
    println!("Got: {}", received);

    // 迭代器接收
    // for received in rx {
    //     println!("Got: {}", received);
    // }
}
```

## 共享状态

### 互斥锁

使用规则：
- 在使用数据之前尝试获取锁
- 处理完被互斥器所保护的数据之后，必须解锁数据

```rust
use std::sync::{Arc, Mutex};
use std::sync::Mutex;
use std::thread;

fn main() {
    // 使用Arc 使用多线程安全的多所有权
    let counter = Arc::new(Mutex::new(0));
    let mut handles = vec![];

    for _ in 0..10 {
        let counter = Rc::clone(&counter);
        let handle = thread::spawn(move || {
        // 阻塞调用，直到拿到锁
            let mut num = counter.lock().unwrap();

            *num += 1;
        // 包含drop 方法，离开作用域自动释放锁
        });
        handles.push(handle);
    }

    for handle in handles {
        handle.join().unwrap();
    }

    println!("Result: {}", *counter.lock().unwrap());
}
```

## 相关文章

- [Rust 程序设计语言: 无畏并发](https://kaisery.github.io/trpl-zh-cn/ch16-00-concurrency.html)
