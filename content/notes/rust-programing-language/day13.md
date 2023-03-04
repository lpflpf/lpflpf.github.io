---
title: "Rust 编程语言 - 智能指针"
date: 2023-03-04
tags:
  - rust
categories:
  - 学习笔记
type: posts
lightgallery: true
description: "智能指针"
keywords: 
  - 学习笔记
  - 智能指针
---
> 智能指针起源于C++，并存在于其他语言。Rust 定义了多种不同的智能指针，并提供了多于引用的额外功能。  
> 普通引用和智能指针：引用是一类只借用数据的指针；智能指针拥有他们指向的数据



## 使用 Box<T>指向堆上的数据 

使用场景：
1. 当有一个在编译时未知大小的类型，而又想要在需要确切大小的上下文中使用这个类型值的时候
2. 当有大量数据并希望在确保数据不被拷贝的情况下转移所有权的时候
3. 当希望拥有一个值并只关心它的类型是否实现了特定 trait 而不是其具体类型的时候

**可存储递归数据**
```rust
enum List {
    // 若不使用Box 则会犹豫无法计算内存大小而报错
    // 通过使用Box 可以计算出，Cons 成员需要 Box指针 + 一个int32 大小的空间
    Cons(i32, Box<List>),
    Nil,
}

use crate::List::{Cons, Nil};

fn main() {
    let list = Cons(1, Box::new(Cons(2, Box::new(Cons(3, Box::new(Nil))))));
}
```

## 使用Deref trait 将智能指针当作常规引用处理

常规引用,类似于指针，可以通过解引用访问数据:
```rust
let x = 5;
let y = &x;
assert_eq!(5, *y); // 解引用
```
Box 也可以使用解引用访问数据:
```rust
let x = 5;
let y = Box::new(x);

assert_eq!(5, x);
assert_eq!(5, *y);
```

### 自定义智能指针

类似于Box 的new方法, 通过 Deref trait 实现智能指针

```rust
use std::ops::Deref;

struct MyBox<T>(T);

impl<T> MyBox<T> {
    fn new(x: T) -> MyBox<T> {
        // 此处是数据的值类型
        MyBox(x)
    }
}

impl<T> Deref for MyBox<T> {
  type Target = T;

  fn deref(&self) -> &Self::Target {
    // 先获取第一个元素的指针，返回后可以通过解引用获取数据
    &self.0
  }
}

// 实现解引用的trait
```

### drop trait 实现清理代码

离开代码作用域时运行的trait方法; 离开作用域时自动执行; 类似于C++析构函数

1. **离开drop调用顺序与定义顺序相反**
2. 不能显示调用 drop 方法; 可以通过 std::mem::drop 强制清理

```rust
struct CustomSmartPointer {
    data: String,
}

// 实现drop trait; 离开作用域时打印数据
impl Drop for CustomSmartPointer {
    fn drop(&mut self) {
        println!("Dropping CustomSmartPointer with data `{}`!", self.data);
    }
}

fn main() {
    let c = CustomSmartPointer {
        data: String::from("my stuff"),
    };
    let d = CustomSmartPointer {
        data: String::from("other stuff"),
    };
    println!("CustomSmartPointers created.");
// 执行顺序, 与定义顺序相反
// CustomSmartPointers created.
// Dropping CustomSmartPointer with data `other stuff`!
// Dropping CustomSmartPointer with data `my stuff`!
}
```

## Rc<T> 引用计数智能指针

多所有权需要显示的使用Rust类型 Rc<T>;对上分配的内存供程序多个部分读取。**单线程场景**

离开作用域会自动减引用计数

```rust
enum List {
    Cons(i32, Rc<List>),
    Nil,
}

use crate::List::{Cons, Nil};
use std::rc::Rc;

fn main() {
    let a = Rc::new(Cons(5, Rc::new(Cons(10, Rc::new(Nil)))));
    // Rc::clone 只是增加引用计数，花费时间短，不进行深拷贝
    let b = Cons(3, Rc::clone(&a));
    let c = Cons(4, Rc::clone(&a));
}
```

## RefCell<T>和内部可变性模式

- 内部可变形是指：允许在不可变引用时，也可以改变数据。
- 通过unsafe代码模糊Rust的可变性和借用规则 （需要手动检查）

### 借用规则检查
*对于Box<T> 借用规则的不可变性在编译时可以确定；而RefCell<T>则在运行时确定（所以在运行时可能引起panic）*
**三者的比较**
- Rc<T> 允许相同数据有多个所有者；Box<T> 和 RefCell<T> 有单一所有者。
- Box<T> 允许在编译时执行不可变或可变借用检查；Rc<T>仅允许在编译时执行不可变借用检查；RefCell<T> 允许在运行时执行不可变或可变借用检查。
- 因为 RefCell<T> 允许在运行时执行可变借用检查，所以我们可以在即便 RefCell<T> 自身是不可变的情况下修改其内部的值。  

**常见用法**

RefCell<T> 与 Rc<T> 结合使用，可以实现可变的多个引用。
```rust
#[derive(Debug)]
enum List {
    Cons(Rc<RefCell<i32>>, Rc<List>),
    Nil,
}

use crate::List::{Cons, Nil};
use std::cell::RefCell;
use std::rc::Rc;

fn main() {
    let value = Rc::new(RefCell::new(5));

    let a = Rc::new(Cons(Rc::clone(&value), Rc::new(Nil)));

    // b, c 都引用了a，
    let b = Cons(Rc::new(RefCell::new(3)), Rc::clone(&a));
    let c = Cons(Rc::new(RefCell::new(4)), Rc::clone(&a));

    *value.borrow_mut() += 10;
    // 最后都看到可变引用 ,a,b,c 中的value值都变成了15
    println!("a after = {:?}", a);
    println!("b after = {:?}", b);
    println!("c after = {:?}", c);
}
```



## 相关文章

- [Rust 程序设计语言](https://kaisery.github.io/trpl-zh-cn/ch15-00-smart-pointers.html)
