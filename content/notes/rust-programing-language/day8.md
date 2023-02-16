---
title: "Rust 编程语言 - 常见集合"
date: 2023-02-16T20:05:08+08:00
draft: true
---

## vector 存储列表

- 类型 Vec<T>
- 新建 ` let v: Vec<i32> = Vec::new();`
  - 由于推断不出类型，所以需要增加类型注解，强化模版类型
  - `vec!` 宏初始化

```rust
let mut v = Vec::new() // 可以推断，所以不需要注解
v.push(123) // 推断为123
```
- 增加元素 `v.push(123)`;

```rust
let v = vec![1, 2, 3, 4, 5];

let third: &i32 = &v[2];
println!("The third element is {third}");

let third: Option<&i32> = v.get(2);
match third {
    Some(third) => println!("The third element is {third}"),
    None => println!("There is no third element."),
}
```
- 获取元素两种方式
  - 一种 [] 直接取值，索引不存在会panic
  - 一种 get 方法，返回 Option 类型，通过match 判断是否存在；不存在返回None
- 当数组是可变的，但是借用给一个不可用的变量，则该数组将变得不可用
- vector 遍历

```rust
// 不可变的访问
let v = vec![100, 32, 57];
for i in &v {
        println!("{i}");
}

// 可变的引用
for i in &mut v {
    *i += 50
        println!("{i}");
}

```

### vector 中存储不同类型

采用枚举类型存储数据. 枚举类型可以标记出最大的存储大小，确保Vec长度是一定的.eg
```rust
enum SpreadsheetCell {
    Int(i32),
    Float(f64),
    Text(String),
}

let row = vec![
    SpreadsheetCell::Int(3),
    SpreadsheetCell::Text(String::from("blue")),
    SpreadsheetCell::Float(10.12),
];
```

## 字符串

本质：字节的集合
### 新建字符串

- 字符串new `String::new()`
- 字符串字面值to_string

```rust
let mut s = String::new()

let data = "initial content"
let s = data.to_string()
// 直接取
let s = "initial content".to_string()
// 等价于
let s = String::from("initial content")
```
- 字符串是u8 编码

### 更新字符串

```rust

let mut s = String::from("foo");
// 追加字面量值
s.push_str("bar");

let mut s1 = String::from("foo");
let s2 = "bar";
s1.push_str(s2); // 并没有获取所有权

// 加号拼接

let s1 = String::from("Hello,");
let s2 = String::from("workd!");

// s1 不能再被使用
let s3 = s1 + &s2; 


// format 

let s1 = String::from("Hello,");
let s2 = String::from("workd!");
let s3 = String::from("toe");

let s = format!("{s1}-{s2}-{s3}");
```

### 索引字符串

**rust 字符串不支持索引**

字符串长度由于u8的原因，取到的值可能不符合预期，所以不支持索引

支持通过range 方式获取字节 `hello[0...4]`

### 遍历字符串方法

```rust
// 按照字符遍历
for c in "Зд".chars() {
    println!("{c}");
}

// 按照字节遍历
for b in "Зд".bytes() {
    println!("{b}");
}
```

## HashMap

### 新建 HashMap

新建和插入
```rust
use std::collections::HashMap;
let mut scores = HashMap::new();

// 插入
scores.insert(String::from("Blue"), 10);
scores.insert(String::from("Yellow"), 50);

let field_name = String::from("Favorite color");
let field_value = String::from("Blue");

// 插入后，二者的所有权不再有效
scores.insert(field_name, field_value)

// 不存在时插入
scores.entry(String::from("Yellow")).or_insert(50);

// 访问
let team_name = String::from("Blue");

// copied 是获取值的引用，若值不存在，则设置为0
let score = scores.get(&team_name).copied().unwrap_or(0);

// 遍历
for (key, value) in &scores {
  println!("{key}: {value}");
}
```

### hash 函数

默认使用 SipHash 确保抵御hash表的拒绝服务攻击；可以更换hasher方法
