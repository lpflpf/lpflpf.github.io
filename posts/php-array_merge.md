---
layout:     post
title:      "PHP array_merge 和 array + array 的区别"
subtitle:   "PHP 函数使用"
date:       2018-06-05
author:     "李朋飞"
header-img: "img/post-bg-2015.jpg"
category:   php
tags:
    - array_merge
---

### array\_merge

[官网说明](http://be2.php.net/manual/zh/function.array-merge.php)

将一个或多个数组的单元合并起来，一个数组中的值附加在前一个数组的后面。返回作为结果的数组。 
如果输入的数组中有相同的字符串键名，则该键名后面的值将覆盖前一个值。然而，如果数组包含数字键名，后面的值将不会覆盖原来的值，而是附加到后面。 
如果只给了一个数组并且该数组是数字索引的，则键名会以连续方式重新索引。


### array + array
[官网说明](http://php.net/manual/zh/language.operators.array.php)

\+ 运算符把右边的数组元素附加到左边的数组后面，两个数组中都有的键名，则只用左边数组中的，右边的被忽略。


### 例子如下

example 1:

```php
$addend = [ 'a' => 'apple', 'b' => 'banana', 'c' => 'cherry' ];
$summand = [ 'a' => 'apricot', 'b' => 'banana', 'd' => 'date' ];

// 非数字键key 相同不做merge，只是追加

print_r(array_merge($addend, $summand));
// key 相同，做覆盖

print_r($addend + $summand);
```

result:

```php
Array
(
     [a] => apricot
     [b] => banana
     [c] => cherry
     [d] => date
)
Array
(
     [a] => apple
     [b] => banana
     [c] => cherry
     [d] => date
)
```

example 2:

```php
$addend = [ 0 => 'apple', 1 => 'banana', 3 => 'cherry' ];
$summand = [ 0 => 'apricot', 1 => 'banana', 2 => 'date' ];

print_r(array_merge($addend, $summand));
print_r($addend + $summand);
```

result:

```
Array
(
     [0] => apple
     [1] => banana
     [2] => cherry
     [3] => apricot
     [4] => banana
     [5] => date
)
Array
(
     [0] => apple
     [1] => banana
     [3] => cherry
     [2] => date
)
```
