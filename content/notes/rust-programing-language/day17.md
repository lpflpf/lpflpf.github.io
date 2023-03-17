---
title: "Rust 编程语言 - 高级特征"
date: 2023-03-15
tags:
  - rust
categories:
  - 学习笔记
type: posts
lightgallery: true
description: "rust 高级特征"
keywords: 
  - 学习笔记
  - 高级特征
---

## 不安全的Rust

**五类可以在unafe中执行，而不能在安全Rust中操作的行为**

1. 解引用裸指针
2. 调用不安全的函数或方法
3. 访问或修改可变静态变量
4. 实现不安全trait
5. 访问union的方法


### 解引用裸指针

裸指针：`*const` `*mut` 是类型一部分
- *const T : 解引用后可变
- *mut T : 解引用后不可变

裸指针与引用和智能指针的区别:
- 允许忽略借用规则，可以同时拥有不可变和可变的指针，或多个指向相同位置的可变指针
- 不保证指向有效的内存
- 允许为空
- 不能实现任何自动清理功能

**可以在安全块内创建裸指针，但是不能从安全块解引用裸指针**
从引用中创建裸指针：

```rust
    let mut num = 5;

    let r1 = &num as *const i32;
    let r2 = &mut num as *mut i32;

    // 不安全块解引用
    unsafe {
        println!("r1 is: {}", *r1);
        println!("r2 is: {}", *r2);
    }

    // 方法的定义和调用
    unsafe fn dangerous() {}

    unsafe {
        dangerous();
    }
```


### 访问和修改静态变量

static 和 const 的区别：
- static 有内存地址空间，const没有
- const 可被随处复制
- static 变量可变
```rust
static mut COUNTER: u32 = 0;

fn add_to_count(inc: u32) {
    unsafe {
        COUNTER += inc;
    }
}

fn main() {
    add_to_count(3);

    unsafe {
        println!("COUNTER: {}", COUNTER);
    }
}
```

## 高级Trait

### 关联类型
关联类型在 trait 定义中指定占位符类型;泛化trait的实现

```rust
pub trait Iterator<T> {
    fn next(&mut self) -> Option<T>;
}
```

### 运算符重载

不支持运算符重载，但可以通过实现运算符相关trait实现重载

```rust
use std::ops::Add;

struct Millimeters(u32);
struct Meters(u32);

impl Add<Meters> for Millimeters {
    type Output = Millimeters;

    fn add(self, other: Meters) -> Millimeters {
        Millimeters(self.0 + (other.0 * 1000))
    }
}
```

### 调用相同名称的方法

```rust
trait Pilot {
    fn fly(&self);
}

trait Wizard {
    fn fly(&self);
}

struct Human;

impl Pilot for Human {
    fn fly(&self) {
        println!("This is your captain speaking.");
    }
}

impl Wizard for Human {
    fn fly(&self) {
        println!("Up!");
    }
}

impl Human {
    fn fly(&self) {
        println!("*waving arms furiously*");
    }
}

fn main() {
    let person = Human;
    // 限定调用的方法
    Pilot::fly(&person); 
    Wizard::fly(&person); 
    person.fly();
}
```

### newtype 类型，隐藏细节

定义新类型，隐藏内部是hashmap的实现细节
```rust
type Wrapper(HashMap<i32, string>)
```

### 类型别名
通过类型别名，减少类型定义的重复,与原类型相同。

```rust
// 减少类型的重复,定义Trunk类型
 type Thunk = Box<dyn Fn() + Send + 'static>;

    let f: Thunk = Box::new(|| println!("hi"));

    fn takes_long_type(f: Thunk) {
        // --snip--
    }

    fn returns_long_type() -> Thunk {
        // --snip--
    }
```

### 闭包的返回

**函数返回闭包，不能直接返回（不知道需要申请多大的空间存储闭包)**
```rust
fn returns_closure() -> Box<dyn Fn(i32) -> i32> {
    Box::new(|x| x + 1)
}
```

### 宏和函数区别

宏：生成代码的代码

1. 函数需要申请参数个数和类型(宏可以接受不定参数)
2. 宏可以在编译器翻译前展开
3. 宏定义通常要比函数定义更难阅读、理解以及维护
4. **在一个文件里调用宏 之前 必须定义它，或将其引入作用域，而函数则可以在任何地方定义和调用**

### 通用元编程

```rust
// vec![1,2,3] 的例子
#[macro_export]
macro_rules! vec {
    ( $( $x:expr ),* ) => {
        {
            let mut temp_vec = Vec::new();
            $(
                temp_vec.push($x);
            )*
            temp_vec
        }
    };
}
// 生成的代码 
{
    let mut temp_vec = Vec::new();
    temp_vec.push(1);
    temp_vec.push(2);
    temp_vec.push(3);
    temp_vec
}
```
### 过程宏

由于无反射，无法在运行时获取类型名称，因此，可以通过过程宏增加注解，实现类型方法

```rust
use proc_macro::TokenStream;
use quote::quote;
use syn;

// 解析语法树，获取类型名称，并打印
#[proc_macro_derive(HelloMacro)]
pub fn hello_macro_derive(input: TokenStream) -> TokenStream {
    // Construct a representation of Rust code as a syntax tree
    // that we can manipulate
    let ast = syn::parse(input).unwrap();

    impl_hello_macro(&ast)
}

fn impl_hello_macro(ast: &syn::DeriveInput) -> TokenStream {
    let name = &ast.ident;
    let gen = quote! {
        impl HelloMacro for #name {
            fn hello_macro() {
                println!("Hello, Macro! My name is {}!", stringify!(#name));
            }
        }
    };
    gen.into()
}

// 使用方
use hello_macro::HelloMacro;
use hello_macro_derive::HelloMacro;

// 引入过程宏
#[derive(HelloMacro)]
struct Pancakes;

fn main() {
    // 直接调用
    Pancakes::hello_macro();
}
```

### 类属性宏

通过宏定义，修改类属性。

```rust
#[route(GET, "/")]
fn index() {
}
// 宏的实现
pub fn route(attr: TokenStream, item: TokenStream) -> TokenStream {
}
```

### 类函数宏

```rust
// sql! 用于检查sql正确性
let sql = sql!(SELECT * FROM posts WHERE id=1);

// 类似宏方法实现的定义
#[proc_macro]
pub fn sql(input: TokenStream) -> TokenStream {
}

```



## 相关文章

- [Rust 程序设计语言: 模式与模式匹配](https://kaisery.github.io/trpl-zh-cn/ch19-01-unsafe-rust.html)
