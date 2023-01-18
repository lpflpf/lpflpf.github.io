---
title:      "PHP singleton"
subtitle:   "PHP 单例模式"
date:       2018-05-30
author:     "李朋飞"
header-img: "intro/runner.jpg"
tags:
    - singleton
    - serialize / unserialize
    - clone

category:
    - php
---

### 单例

  单例模式，希望在程序上下文中，仅对对象做一次实例化。

### 问题
  - clone 会引起出现单例对象有多个实例

### 避免方式
  - clone 由于clone时，会调用对象的`__clone` magic method. 因此，可以将`__clone` 设置为私有，使clone失效。

### 单例模式
```
    class Singleton {
        private static $instance = NULL;

        private function __construct() { }

        private function __clone() { }

        public static function getInstance() {
            if (NULL === self::$instance) {
                self::$instance = new self();
            }
            return self::$instance;
        }
    }
```

### unserialize

将Singleton 将对象序列化，再进行反序列化。可以构造新的单例对象。这种情况下，鸟哥给出的解决方案是：(使用`__wakeup()` 方法)[http://www.laruence.com/2011/03/18/1909.html]。
但是，通过对比发现，这个代码其实会有问题：
```
    <?php
    class Singleton {
        private static $instance = NULL;
        private $data = '';

        private function __construct() { }

        private function __clone() { }

        public  function __wakeup() {
            self::$instance = $this;
        }

        public static function getInstance() {
            if (NULL === self::$instance) {
                self::$instance = new self();
            }
            return self::$instance;
        }

    }

    $a = Singleton::getInstance();
    $b = unserialize(serialize($a));
    
    // false
    var_dump($b === $a);
```

**因为unserialize 其实会实例化一个单例对象，和原来实例化的单例对象不是一个对象。因此，会引起dump 不一致的情况。这个暂时无法避免。**
