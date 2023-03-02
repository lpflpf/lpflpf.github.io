---
title: "Rust 编程语言 - 函数式语言功能：闭包"
date: 2023-02-23
tags:
  - rust
categories:
  - 学习笔记
type: posts
lightgallery: true
description: "rust 迭代器与闭包"
keywords: 
  - 学习笔记
  - 迭代器与闭包
  - 迭代器
---

## 闭包：可以捕获环境的匿名函数

- 使用 `|| 方法` 标记 `||` 中间可以填写参数

闭包的case1:

```rust
#[derive(Debug, PartialEq, Copy, Clone)]
enum ShirtColor {
    Red,
    Blue,
}

struct Inventory {
    shirts: Vec<ShirtColor>,
}

impl Inventory {
    fn giveaway(&self, user_preference: Option<ShirtColor>) -> ShirtColor {
      // || self.most_stocked() 是闭包表达式
        user_preference.unwrap_or_else(|| self.most_stocked())
    }

    fn most_stocked(&self) -> ShirtColor {
        let mut num_red = 0;
        let mut num_blue = 0;

        for color in &self.shirts {
            match color {
                ShirtColor::Red => num_red += 1,
                ShirtColor::Blue => num_blue += 1,
            }
        }
        if num_red > num_blue {
            ShirtColor::Red
        } else {
            ShirtColor::Blue
        }
    }
}

fn main() {
    let store = Inventory {
        shirts: vec![ShirtColor::Blue, ShirtColor::Red, ShirtColor::Blue],
    };

    let user_pref1 = Some(ShirtColor::Red);
    let giveaway1 = store.giveaway(user_pref1);
    println!(
        "The user with preference {:?} gets {:?}",
        user_pref1, giveaway1
    );

    let user_pref2 = None;
    let giveaway2 = store.giveaway(user_pref2);
    println!(
        "The user with preference {:?} gets {:?}",
        user_pref2, giveaway2
    );
}
```

闭包case2:
**闭包不一定需要fn，比函数要灵活**

```rust
 let expensive_closure = |num: u32| -> u32 {
        println!("calculating slowly...");
        thread::sleep(Duration::from_secs(2));
        num
  };

// 函数
fn  add_one_v1   (x: u32) -> u32 { x + 1 }
// 闭包
let add_one_v2 = |x: u32| -> u32 { x + 1 };
// 省略类型注解
let add_one_v3 = |x|             { x + 1 };
// 省略可选的大括号
let add_one_v4 = |x|               x + 1  ;
```

**同一个闭包不能推断用不一样的类型**

## 闭包捕获引用或者移动所有权

- 不可变借用
不可变引用，在闭包定义后，使用前还可以访问变量
```rust
fn main() {
    let list = vec![1, 2, 3];
    println!("Before defining closure: {:?}", list);

    let only_borrows = || println!("From closure: {:?}", list);

    println!("Before calling closure: {:?}", list);
    only_borrows();
    println!("After calling closure: {:?}", list);
}
```
- 可变借用
可变引用，在闭包定义后，使用前不能再访问变量；闭包使用后可以继续访问
```rust
fn main() {
    let mut list = vec![1, 2, 3];
    println!("Before defining closure: {:?}", list);

    let mut borrows_mutably = || list.push(7);

    borrows_mutably();
    println!("After calling closure: {:?}", list);
}
```
- 获取所有权
可以使用move 强制使用所有权
```rust
use std::thread;

fn main() {
    let list = vec![1, 2, 3];
    println!("Before defining closure: {:?}", list);

  // 依赖的变量 移动所有权
    thread::spawn(move || println!("From thread: {:?}", list))
        .join()
        .unwrap();
}
```

### 将被捕获的值移出闭包和 Fn trait

三个trait
- `FnOnce` 至少可以调用一次闭包方法
- `FnMut`  不会将捕获的值移除闭包, 但修改捕获的值，可以调用多次
- `Fn` 既不将被捕获的值移出闭包体也不修改被捕获的值的闭包

**例如** Option<T> 的 unwrap_or_else 方法：
```rust
impl<T> Option<T> {
    pub fn unwrap_or_else<F>(self, f: F) -> T
    where
        F: FnOnce() -> T // F 必须要实现 FnOnce Trait
    {
        match self {
            Some(x) => x,
            None => f(),
        }
    }
}
```
## 使用迭代器处理元素序列

1. rust 迭代器是惰性的,在调用时才会有效果
```rust
let v1 = vec![1,2,3];
let v1_iter = v1.iter();
for val in v1_iter {
    println!("Got: {}", val);
}
```

2. 迭代器的实现，其实是实现了一个next 的trait
```rust
pub trait Iterator {
    type Item;

    fn next(&mut self) -> Option<Self::Item>;

    // 此处省略了方法的默认实现
}
```
3. `iter` 返回不可变引用； `into_iter` 获取所有权; 迭代可变引用 `iter_mut`

## 消费迭代器的方法

调用next方法的方法，被称为消费迭代器，调用完成后，迭代器将不可用。（获取所有权后，所有权不可用）
```rust
    #[test]
    fn iterator_sum() {
        let v1 = vec![1, 2, 3];

        let v1_iter = v1.iter();

        let total: i32 = v1_iter.sum();

        assert_eq!(total, 6);
    }
```

## 产生其他迭代器的方法

Iterator trait 中定义了另一类方法，被称为 迭代器适配器;

```rust
let v1: Vec<i32> = vec![1, 2, 3];
// map 是迭代适配器，为了防止 x 获取所有权但是没 做任何事情，使用collect 做收集
let v2: Vec<_> = v1.iter().map(|x| x + 1).collect();
assert_eq!(v2, vec![2, 3, 4]);
```

**迭代器是零成本抽象，迭代器的成本可能比循环还要高**
1. 可能会做循环展开
2. 可能不需要做边界检查
3. 需要的系数存在了寄存器中，访问速度更快




- [文章链接](https://kaisery.github.io/trpl-zh-cn/ch13-00-functional-features.html)
- [文章链接](https://kaisery.github.io/trpl-zh-cn/ch13-02-iterators.html)