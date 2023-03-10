---
title: "Rust 编程语言 - 面向对象特质"
date: 2023-03-10
tags:
  - rust
categories:
  - 学习笔记
type: posts
lightgallery: true
description: "rust 面向对象"
keywords: 
  - 学习笔记
  - 面向对象
---

## 面向对象语言特征

1. 对象包含数据和行为
> 四人帮解释：面向对象的程序是由对象组成的。一个对象包含数据和操作这些数据的过程。这些过程通常被称为 方法 或 操作。
> 通用解释：一个对象包含了数据和对数据的操作过程,即为面向对象
2. 封装 隐藏细节

使用 `pub` 关键字标记是否公有

```rust
// 公有struct
pub struct AveragedCollection {
    // 内部成员私有
    list: Vec<i32>,
    average: f64,
}

impl AveragedCollection {
    // 公有方法
    pub fn add(&mut self, value: i32) {
        self.list.push(value);
        self.update_average();
    }

    pub fn remove(&mut self) -> Option<i32> {
        let result = self.list.pop();
        match result {
            Some(value) => {
                self.update_average();
                Some(value)
            }
            None => None,
        }
    }

    pub fn average(&self) -> f64 {
        self.average
    }

    // 私有方法
    fn update_average(&mut self) {
        let total: i32 = self.list.iter().sum();
        self.average = total as f64 / self.list.len() as f64;
    }
}

```
3. 继承 
rust 没有继承, 但可以通过trait 实现共享
4. 多态
使用  trait bounds 约束 (类似于golang interface{}, 方法的约束)

## 不同类型的值的trait对象

### 定义通用行为的trait

鸭子类型：实现了trait 定义的方法，你就实现了某个trait

一个GUI接口的抽象
```rust
// trait 的定义，类似go接口
pub trait Draw {
    fn draw(&self);
}

pub struct Screen {
    // 实现了Trait Draw 的Vec, 比模版更通用
    pub components: Vec<Box<dyn Draw>>,
}

impl Screen {
    pub fn run(&self) {
        for component in self.components.iter() {
            component.draw();
        }
    }
}

// button 是对Trait 的一个实现
pub struct Button {
    pub width: u32,
    pub height: u32,
    pub label: String,
}

// 实现 Draw triat 需要的方法
impl Draw for Button {
    fn draw(&self) {
        // code to actually draw a button
    }
}

// selectBox 也实现了Draw trait
struct SelectBox {
    width: u32,
    height: u32,
    options: Vec<String>,
}

impl Draw for SelectBox {
    fn draw(&self) {
        // code to actually draw a select box
    }
}

fn main() {
    let screen = Screen {
        components: vec![
            Box::new(SelectBox {
                width: 75,
                height: 10,
                options: vec![
                    String::from("Yes"),
                    String::from("Maybe"),
                    String::from("No"),
                ],
            }),
            Box::new(Button {
                width: 50,
                height: 10,
                label: String::from("OK"),
            }),
        ],
    };

    screen.run();
}
```
### 动态分发

由于对于trait，无法对对象进行预测，所以方法调用是动态分发的。性能上需要有些取舍

## 相关文章

- [Rust 程序设计语言: 无畏并发](https://kaisery.github.io/trpl-zh-cn/ch17-00-oop.html)
