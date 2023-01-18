---
title: 由ctype_digit 引起的一个问题
date:       2018-07-26
tags:
    - PHP 开发
    - ctype函数簇

category:
    - php
---
ctype_digit 问题说明。
<!--more-->

### 问题说明

最近发现，代码中有先人使用ctype_digit函数判断缓存时间，若返回为true，则设置redis Key的生命周期，否则不设置超时时间。

为了使数据能快速更新，于是有人修改了缓存时间从300 -> 50,导致了数据不再更新。原因就在于ctype_digit 函数的使用方式有问题。


### ctype 的使用
ctype 用于检测字符串的相关类型

- ctype_alnum 是否为数字或者字母, 是返回TRUE，否则返回false
- ctype_alpha 做纯字符检测
- ctype_cntrl 做控制字符检测, 换行、缩进、空格等
- ctype_digit 做纯数字检测
- ctype_graph 可打印字符串检测，空格除外 
- ctype_lower 做小写字符检测 
- ctype_print 做可打印字符检测 
- ctype_punct 检测可打印的字符是不是不包含空白、数字和字母
- ctype_space 做空白字符检测 
- ctype_upper  做大写字母检测 
- ctype_xdigit 检测字符串是否只包含十六进制字符

  **此类函数若给出的是-128到255之间（含）的整数，会被解释为该值对应的ASCII（负值将加上256 以支持扩展ASCII字符）。其他将呗认为是字符串**

