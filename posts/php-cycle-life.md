---
layout:     post
title:      "PHP cycle life"
subtitle:   "运行流程解析"
date:       2018-06-06
author:     "李朋飞"
header-img: "img/post-bg-2015.jpg"
category:   
    - php
tags:
    - cycle life
    - Request 处理
    - PHP Source Analysis
---

### php 运行阶段

  - 开始阶段
      - 模块初始化 MINIT (module init)
          这个阶段，将对每个扩展的PHP_MINIT_FUNCTION函数执行。一般执行如下操作：
          1. INI 配置文件的注册  REGISTER_INI_ENTRIES
          2. 定义该扩展实现的类, Interface等
          3. 定义的const变量

      - 模块激活   RINIT (Request init)   
          每个请求进入时，将调用每个扩展的PHP_RINIT_FUNCTION。一般有如下需求会调用该方法:
          1. 重置之前的请求, 例如 spl 扩展。
          2. 通过请求数据，初始化模块的参数。 例如 mbstring 扩展。

  - 运行阶段
      - 进入PHP文件执行
  - 结束阶段
      - 停用模块   RSHUTDOWN ( Request shutdown)
        与 RINIT 相对应
      - 关闭模块   MSHUTDOWN (module shutdown)  
        与 MINIT 相对应

## 不同的PHP运行环境，PHP的生命周期不同

### php 命令行模式

![](php_cycle_life_cli.jpg)

### php Multi Process 模式

![multi Process](php_cycle_life_multi_process.jpg)

### PHP Multi Threaded 模式
![multi Threaded](php_cycle_life_multithreaded.png)

          
### 其他一些函数
  1. PHP_GINIT_FUNCTION  全局变量初始化
  2. PHP_GSHUTDOWN_FUNCTION  
  3. PHP_MINFO_FUNCTION 设置INI 文件中模块的信息, phpinfo 时打印的数据
  4. CG  Complier Global
  5. EG  Executor Global
  6. PG  PHP Core Global
  7. SG  SAPI Global
