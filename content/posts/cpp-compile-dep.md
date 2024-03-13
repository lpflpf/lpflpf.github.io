---
title: "编译依赖问题的一个坑"
date: 2024-03-13T09:01:47+08:00
draft: true
tags:
    - c++
    - 编译原理
---

记录一次gcc，编译静态库依赖顺序导致编译失败的问题。
<!--more-->

假设有个cpp，依赖两个静态库。静态库代码如下：

```cpp
// a.cpp
// g++ -c a.cpp
// ar cr a.a a.o
// 
void a() {}
```

```cpp
// b.cpp 依赖 a.cpp
// g++ -c b.cpp
// ar cr b.a b.o
extern void a();
void b() {a();}
```

```cpp
// main.cpp 依赖 b.cpp 
extern void b();
int main()
{
    b(); // 调用b.cpp中的b()
    return 0;

}
```

针对main.cpp 的编译，就会出现问题。
在GNU 的g++ 版本中，必须要使用 `g++ main.cpp b.a a.a `； 如果使用 `g++ main.cpp a.a b.a `会报如下错误:

```plain
b.a(b.o)：在函数‘b()’中：
b.cpp:(.text+0x5)：对‘a()’未定义的引用
```

而在clang 的g++版本中，则二者的顺序没有依赖性，都是ok的。

总结起来，对于静态链接库的顺序而言，GUN 的g++版本，前面的库依赖后面库的实现。 否则会报`未定义的引用`。

### 如果有问题，如何解决

- 需要的lib包，可以写多次，解决依赖问题
- 调整依赖顺序
- [包互相依赖的问题，通过 `-( -) ` 包起来，多次引入]((https://stackoverflow.com/questions/45135/why-does-the-order-in-which-libraries-are-linked-sometimes-cause-errors-in-gcc))

### 动态链接库有此问题吗

动态链接库会自动调整顺序，会进行自动排序。

### 为什么需要有依赖性

主要为了改善链接器的性能，按顺序来就不用重复扫描库文件了。 每个库只需要扫描一次。
链接器在工作过程中，维护3个集合：需要参与连接的目标文件集合E、一个未解析符号集合U、一个在E中所有目标文件定义过的所有符号集合D。目标文件按照顺序解析, 因此不会把没有用到的.a库里的.o 文件加入U集合导致, 编译错误。

## 学习文章

- [stackflow](https://stackoverflow.com/questions/17669941/g-the-order-of-static-library-matters)
- [link-error-question](https://stackoverflow.com/questions/45135/why-does-the-order-in-which-libraries-are-linked-sometimes-cause-errors-in-gcc)
- [依赖顺序的原因](https://www.zhihu.com/question/387001677/answer/1146215465?utm_id=0)