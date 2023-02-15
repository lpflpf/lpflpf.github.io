---
title: "Rust 编程语言 - 使用包、Crate 和模块管理不断增长的项目"
date: 2023-02-13
tags:
  - rust
categories:
  - 学习笔记
type: posts
lightgallery: true
keywords: 
  - 学习笔记
  - 作用域
---

## 系统模块
- 包（Packages)：Cargo的功能，用于构建测试和分享crate
- Crates： 一个模块的树形结构,形成库或二进制项目
- 模块（Modules) 和 use：允许控制作用域和路径的私有行
- 路径(path): 一个命名，例如结构体、函数或者模块等项的方式

### 包和crate

包中默认查找两个rs; src/main.rs, src/lib.rs; 确认生成两个crate.(一个二进制，和一个库)

### 定义模块来控制作用域和私有性

- 从crate根节点寻找需要被编译的文件 (main.rs, lib.rs)
- 在crate 文件中声明 模块 (`mod xxx`)
- 模块中可以定义子模块
- 隐私规则允许下，可以任意地方引入模块代码
- 一个模块代码默认对父模块私有，可以在声明是 `pub mod xxx` 描述为公有模块
- use 模块代码快捷方式

## 引用项目模块的路径

- 可采用绝对路径、相对路径引用模块
- 更倾向于绝对路径，把代码定义和项调用各自独立地移动是比较常见的
- 要对外暴露方法，必须模块和方法都是pub

### super 的使用

- super 可以访问父组件，类似目录里面的`..`
- 使用super 可以免去绝对路径里面的变动

### 结构体的共有私有

- 结构体公有，字段不设置，还是私有的
- 结构体常规，不会将字段值设为共有
- 枚举公有，则所有成员都是公有

### 使用 use 关键字将路径引入作用域

- use 创建了作用域上的软链，对子模块不生效
- 一般不会use 方法名，因为在代码上下文不好定位函数来源
- `as` 将引用的模块 重命名，解决重名问题

### 消除大量use 行

- 嵌套路径 `use std::{cmp::Ordering, io};`
- glob运算符 快速引入 `use std::collections::*;` **多用于测试**

## 将模块拆分成多个文件

- 文件名和模块名一致 `use create::front_of_house::hosting` 则到 `front_of_house.rs` 中先查找; 发现`front_of_house.rs`为 `pub mod hosting` ，按照目录结构，hosting.rs 放在 front_of_house 文件中
- 也可以在 front_of_house 中定义一个 mod.rs，(老风格)