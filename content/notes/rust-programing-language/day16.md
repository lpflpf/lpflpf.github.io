---
title: "Rust 编程语言 - 模式与模式匹配"
date: 2023-03-14
tags:
  - rust
categories:
  - 学习笔记
type: posts
lightgallery: true
description: "rust 模式与模式匹配"
keywords: 
  - 学习笔记
  - rust 模式与模式匹配
---

## 会用到模式的位置

### match 分支
**match 分支必须是穷尽的**

```rust
match x { // 对枚举类型 Option<i32> 值的枚举
    Some(i) => Some(i+1),
    None => None
}
```

### if let 条件表达式

```rust

let age: Result<u8, _> = "34".parse()

if let Ok(age) = age {
    // age 重新复制为 u8
}
```

### while let 条件循环

```rust
let mut stack = Vec::new()
stack.push(1);
stack.push(2);
stack.push(3);
// 若匹配为Some 类型，则while继续执行
while let Some(top) = stack.pop() {
    // top 为值类型
}
```

### for 循环

```rust
let v = vec!['a', 'b', 'c'];

// 索引与值的匹配
for (index, value) in v.iter().enumerate() {
    println!("{} is at index {}", value, index);
}
```

### let 语句

```rust
let (x,y,z) = (1,2,3) // 赋值匹配
```

### 函数参数

```rust
fn print_coordinates(&(x, y): &(i32, i32)) {
    // 匹配后，x, y 分别被赋值
    println!("Current location: ({}, {})", x, y);
}

fn main() {
    let point = (3, 5);
    print_coordinates(&point);
}
```

## Refutability (可反驳性)：模式是否会匹配失效
- 某些可能的值进行匹配会失败的模式被称为是 可反驳的
例如: `if let Som(x) = a_Value`
- 能匹配任何传递的可能值的模式被称为是 不可反驳的（irrefutable）
例如：`let x = 5` 不会失败，不可反驳

因此，对于match匹配分支，必须使用可反驳模式。对于if等使用不可反驳分支时，会warning

## 所有的模式语法

### 匹配字面值
```rust
let x = 1;
match x {
    1 => println!("one"),
    2 => println!("two"),
    _ => println!("anything"), // 其他
}
```

### 匹配命名变量
```rust
let x = Some(5);
let y = 10;

match x {
    Some(50) => println!("Got 50"),
    Some(y) => println!("Matched, y = {y}")
    _ => println!("Default case, x = {:?}", x)
}
```

### 多个模式匹配
```rust
let x = 1;

match x {
    1|2 => println!("one or two"),
    3 => println!("three"),
    _ => println!("anything"), // 其他
}

// 通过 ..= 匹配值范围

let x = 5

match x {
    1 ..= 5 => println!("one through five"),
    _ => println!("anything else"),
}
```
### 解构结构体

```rust
struct Point {
    x: i32,
    y: i32,
}

fn main(){
    let p = Point{x:0, y:7}
    let Point{x:a, y:b} = p
    // 或者
    let Point{a, b} = p
    assert_eq(0,a);
    assert_eq(7,b);


    // 模式匹配
    match p {
        Point {x, y: 0} => println!("On the x asis at {x}")
        Point {x: 0, y} => println!("On the y asis at {y}")
        Point {x, y} => println!("On neither axis: ({x}, {y})")
    }
}
```

### 解构枚举
```rust
enum Message {
    Quit,
    Move { x: i32, y: i32 },
    Write(String),
    ChangeColor(i32, i32, i32),
}

fn main() {
    let msg = Message::ChangeColor(0, 160, 255);

    match msg {
        Message::Quit => {
            println!("The Quit variant has no data to destructure.");
        }
        Message::Move { x, y } => {
            println!("Move in the x direction {x} and in the y direction {y}");
        }
        Message::Write(text) => {
            println!("Text message: {text}");
        }
        Message::ChangeColor(r, g, b) => {
            println!("Change the color to red {r}, green {g}, and blue {b}",)
        }
    }
}
```


### 解构嵌套的结构体和枚举

```rust
enum Color {
    Rgb(i32, i32, i32),
    Hsv(i32, i32, i32),
}

enum Message {
    Quit,
    Move { x: i32, y: i32 },
    Write(String),
    ChangeColor(Color),
}

fn main() {
    let msg = Message::ChangeColor(Color::Hsv(0, 160, 255));

    match msg {
        // 嵌套枚举值的match
        Message::ChangeColor(Color::Rgb(r, g, b)) => {
            println!("Change color to red {r}, green {g}, and blue {b}");
        }
        Message::ChangeColor(Color::Hsv(h, s, v)) => {
            println!("Change color to hue {h}, saturation {s}, value {v}")
        }
        _ => (),
    }
}
```

### 解构结构体和元组

** 支持对嵌套元组的解构
```rust
let ((feet, inches), Point { x, y }) = ((3, 10), Point { x: 3, y: -10 });
```

### 使用 `_` 忽略整个值
```rust
fn foo(_: i32, y: i32) {
    println!("This code only uses the y parameter: {}", y);
}

fn main() {
    foo(3, 4);
}
```

### 忽略部分值

```rust
let numbers = (2, 4, 8, 16, 32);

match numbers {
    (first, _, third, _, fifth) => {
        println!("Some numbers: {first}, {third}, {fifth}")
    }
}
```

### 忽略未使用变量 
```rust
fn main() {
    let _x = 5; // 变量前使用
    let y = 10;
}
```

### `..` 忽略剩余值 

```rust
struct Point {
    x: i32,
    y: i32,
    z: i32,
}

let origin = Point { x: 0, y: 0, z: 0 };

match origin {
    //.. 使用必须无歧义
    Point { x, .. } => println!("x is {}", x),
}
```

### 匹配守卫提供额外条件

```rust

let num = Some(4);

match num {
    // 当Some(x) 枚举模式，并且 x 为偶数
    Some(x) if x % 2 == 0 => println!("The number {} is even", x),
    Some(x) => println!("The number {} is odd", x),
    None => (),
}
```

### `@`运算符绑定

允许在创建一个存放值的变量的同时，测试其值是否匹配模式

```rust
enum Message {
    Hello { id: i32 },
}

let msg = Message::Hello { id: 5 };

match msg {
    Message::Hello {
        id: id_variable @ 3..=7, // 创建变量 id_variable ，并看是否在 3..=7 之间
    } => println!("Found an id in range: {}", id_variable),
    Message::Hello { id: 10..=12 } => {
        println!("Found an id in another range")
    }
    Message::Hello { id } => println!("Found some other id: {}", id),
}
```



## 相关文章

- [Rust 程序设计语言: 模式与模式匹配](https://kaisery.github.io/trpl-zh-cn/ch18-00-patterns.html)
