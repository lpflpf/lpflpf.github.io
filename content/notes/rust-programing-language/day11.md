---
title: "Rust 编程语言 - 编写自动化测试"
date: 2023-02-22
tags:
  - rust
categories:
  - 学习笔记
type: posts
lightgallery: true
description: "rust 自动化测试"
keywords: 
  - 学习笔记
  - 自动化测试
---

## 如何编写测试

1. 需要标记注解 `#[test]`
2. `assert_eq` 宏 和 `assert_ne` 宏 测试相等
3. `assert` 宏自定义断言
4. `should_panic` 注解预测panic
5. 还可以通过Result<T, E> 方式测试

## 控制测试如何运行 

1. 默认并行测试，可以通过 `--test-threads=1` 指定但县城测试
2. 显示函数输出 `--show-output`
3. 可以指定运行部分测试
4. `[ignore]` 注解忽略测试

## 测试的组织

### 单元测试

1. 单元测试，在模块中，指定 `[cfg(test)]` 注解，可以在编译时忽略test包，加快变异，减少编译结果大小
2. 通过子模块引用父模块的方法，测试私有函数 
```rust
pub fn add_two(a: i32) -> i32 {
    internal_adder(a, 2)
}

fn internal_adder(a: i32, b: i32) -> i32 {
    a + b
}

#[cfg(test)]
mod tests {
    // 引入父模块的私有方法
    use super::*;

    #[test]
    fn internal() {
        assert_eq!(4, internal_adder(2, 2));
    }
}
```

### 集成测试

1. 在src 同级创建 `tests` 目录，目录结构如下：
```
adder
├── Cargo.lock
├── Cargo.toml
├── src
│   └── lib.rs
└── tests
    └── integration_test.rs
```
2. 可以支持多模块