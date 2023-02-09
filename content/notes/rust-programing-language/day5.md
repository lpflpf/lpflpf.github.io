---
title: "使用结构体组织相关联的数据"
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


## 定义 

```rust
struct User {
    active: bool,
    username: String,
    email: String,
    sign_in_count: u64,
}
```

## 初始化

```rust
    let mut user1 = User {
        active: true,
        username: String::from("someusername123"),
        email: String::from("someone@example.com"),
        sign_in_count: 1,
    };

// 字段初始化简写
fn build_user(email: String, username: String) -> User {
    User {
        active: true,
        username,
        email,
        sign_in_count: 1,
    }
}

// user2 中的其他字段从user1中获取
// String 类型的数据被移动,user1 中不能再使用
let user2 = User {
        email: String::from("another@example.com"),
        ..user1 // 必须放最后
};


// 元组结构体 
struct Color(i32, i32, i32);
struct Point(i32, i32, i32);

// 类单元结构体,实现 trait
struct AlwaysEqual;
```

## case

```rust
// 标记没有实现
#[derive(Debug)]
struct Rectangle {
    width: u32,
    height: u32,
}

fn main() {
    let rect1 = Rectangle {
        width: 30,
        height: 50,
    };
     println!(
        "The area of the rectangle is {} square pixels.",
        area(&rect1)
    );
}

fn area(rectangle: &Rectangle) -> u32{
    rectangle.width * rectangle.height
}
```

> dbg! 可以打印所有权, 重要

## 方法

```rust
#[derive(Debug)]
struct Rectangle {
    width: u32,
    height: u32,
}

impl Rectangle {
    // 第一个参数总是 &self 代表结构体的引用
    // 需要需要修改 &mut self
    // 内部不需要解引用
    fn area(&self) -> u32 {
        self.width * self.height
    }
}

fn main() {
    let rect1 = Rectangle {
        width: 30,
        height: 50,
    };

    println!(
        "The area of the rectangle is {} square pixels.",
        rect1.area()
    );
}

impl Rectangle {
    // 不一定使用self, 关联函数
    fn square(size: u32) -> Self {
        Self {
            width: size,
            height: size,
        }
    }
}
```