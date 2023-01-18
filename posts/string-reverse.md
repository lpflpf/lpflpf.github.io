---
title:  字符串反转的面试题，你会吗
date: 2020-04-24 16:03:00
tags:
    - interview
---

字符串的面试题，看能到达那个阶段。

<!--more-->

## 第一阶段

不用申请内存空间，把一个字符串做反正操作。

比如说：
str="abcdefg"
res="gfedcba"

这个比较简单，只要做前后字符交换就可以了

```go
func reverse(str []byte){
    i := 0
    j := len(str) - 1

    for i < j {
        str[i], str[j] = str[j], str[i]
        i ++
        j --
    }
}
```

## 第二阶段

不用申请内存，如何把每个单词做反转,假设单词中间只有一个空格

比如说：
str = "php is the best programing language in the world"
res = "php si eht tseb gnimargorp egaugnal ni eht dlrow"

```go
func reverse(str string) {
    i := 0
    k := 0

    reverse1 = func(str []byte, begin int, end int){
        for begin < end {
            str[begin], str[end] = str[end], str[begin]
            begin ++
            end --
        }
    }

    for i = 0; i < len(str); i ++ {
        if str[i] == ' ' {
            reverse1(str, k, i - 1)
            k = i + 1
        }
    }
}
```

## 第三阶段

不用申请内存，如何把一组单词做反转。

比如说：
str = "php is the best programing language in the world"
res = "world the in language programing best the is php"

这个略有难度，但是只需要在第二阶段的接触上加一行代码就可以做到了。

```go
func reverse(str string) {
    i := 0
    k := 0

    reverse1 = func(str []byte, begin int, end int){
        for begin < end {
            str[begin], str[end] = str[end], str[begin]
            begin ++ 
            end --
        }
    }

    reverse1 (str, 0, len(str) - 1)
    for i = 0; i < len(str); i ++ {
        if str[i] == ' ' {
            reverse1(str, k, i - 1)
            k = i + 1
        }
    }
}
```

