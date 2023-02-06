---
title: "Rust 编程语言 - 常见编程概念"
date: 2023-02-02
tags:
  - rust
categories:
  - 学习笔记
type: posts
---

> 阅读[Rust程序设计语言](https://kaisery.github.io/trpl-zh-cn/title-page.html)笔记

# 变量和可变性

1. 变量默认是不可改变的(immutable)
2. 可变变量，需要加 mut 修饰 `let mut x = 5`
3. 常量 使用 const 定义
   1. `const THREE_HOURS_IN_SECONDS: u32 = 60 * 60 * 3;`
   2. 需要申明常量类型
   3. 常量只能被设置为常量表达式，不能是运行时计算出的值 (和变量区别)
4. 隐藏，变量遮蔽。重新申明变量，前面申明的变量会隐藏
   1. 隐藏时，可以改变变量类型

# 数据类型

1. 指定数据类型，或者可以推断出数据类型；
2. 标量类型：整型，浮点型，布尔型，字符型
3. 整形
   1. 有符号，无符号
   2. 8，16，32，64，128，arch 几种长度
   3. 支持下划线分隔符
   4. 整形溢出，debug 会panic；release 会忽略高位
   5. 数字可以加后缀 `123u16` 16位的数字 123
4. 浮点型
   1. f32,f64;
   2. IEE-754
5. 数值运算
   1. 整数除法，四舍五入
6. bool true|false
7. 字符类型
   1. 四个字节，一个Unicode标量
8. 复合类型
   1. 元组
      1. 申明后长度不会增大缩小；类型不必相同
      2. 通过模式匹配解构(destructuring)元组值
      3. 通过(.) 加索引访问
      4. `let x: (i32, i64, f32) = (100, 0, .3)`
   2. 数组
      1. 长度固定；类型必须相同；
      2. 不固定长度，需要使用vector 类型（标准库提供）
      3. `let a = [1,2,3,4,5]`
      4. `let a: [i32; 5] = [1,2,3,4,5];`  指定类型
      5. `let a = [3;5]` 创建5个3
      6. 访问超出界限，会panic

# 函数

1. 加分号是语句，不加分号是表达式(又返回值)

```rust
fn a_func(x: i32){
   println!("hello world");
}

// 返回值类型是i32，返回数据是5
fn five() -> i32 {
   5
}
```

# 注释

1. `// 注释`
2. `/// 文档注释`

# 控制流

1. if 语句
  1. 条件不用加括号
  2. 必须bool类型，不能自动转换
  3. 可以使用let语句赋值

```rust
// if 不用加括号
// 必须是bool类型，不会自动转换
if number < 5 {
   println!("")
}else if number % 3 == 0 {
   println!("")
}else {
   println!("")
}

// if else 返回类型必须相同
let number = if condition {5} else {6};
```
2.  循环 (loop while/for)
   1. break 中断； 可以带返回值中断
```rust
let result = loop {
   counter += 1;

   if counter == 10 {
      break counter * 2; // result 赋值为20
   }
}
```
   2. 多个循环之间消除歧义

   标签 'counting_up (这个好诡异)
```rust
fn main() {
    let mut count = 0;
    'counting_up: loop {
        println!("count = {count}");
        let mut remaining = 10;

        loop {
            println!("remaining = {remaining}");
            if remaining == 9 {
                break;
            }
            if count == 2 {
                break 'counting_up;
            }
            remaining -= 1;
        }

        count += 1;
    }
    println!("End count = {count}");
}
```
   3. while 条件循环

```rust
while index < 5 {
   println!("the valeue is: {}", 123)
   index += 1;
}
```
   4. for 循环

```rust
for number in (1..4).rev() { // rev 反转; 遍历 1到4
   println!("{number}")
}
```


