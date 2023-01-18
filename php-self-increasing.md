---
title: php 自增运算符
date: 2019-06-25 10:29:43
tags:
  - PHP 开发
  - PHP Source Analysis
category:
    - php
---

php 的一些小众的用法，很多php老司机，使用时也会出问题。 

<!--more-->

今天就聊一聊php的自增运算符。

### bool 值
   对于bool值无效。

```
   # php -r '$a=false; $a++; var_dump($a);';
   bool(false)
```

### null 值
   null 值，自增后为整型1.
```
  # php -r '$a=null; $a++; var_dump($a);';
  int(1)
```

### 数字运算
  - 正常范围的整数：

  ```shell
  #  php -r '$a=1; $a++; var_dump($a);';
  int(2)
  ```
  - 最大值的整数,整数直接变成浮点数:

  ```shell
    # php -r '$a=9223372036854775807; $a++; var_dump($a);'
    float(9.2233720368548E+18)
    # php -r '$a=9223372036854775806; $a++; var_dump($a);'
    int(9223372036854775807)
  ```
  - 浮点数的计算:
  若在精度范围内，则自增加1，若不在精度范围内，则忽略。

### 字符运算
#### 继承自perl的字符自增运算符。
  - 以字符结尾

  ```shell
    # php -r '$a="a"; $a++; var_dump($a);';
    string(1) "b"
    # php -r '$a="z"; $a++; var_dump($a);';
    string(2) "aa"
    # php -r '$a="A"; $a++; var_dump($a);';
    string(1) "B"
    # php -r '$a="Z"; $a++; var_dump($a);';
    string(2) "AA"
    # php -r '$a="zzz"; $a++; var_dump($a);';
    string(4) "aaaa"
  ```
  - 数字结尾

  ```shell
  # php -r '$a="Z1"; $a++; var_dump($a);';
  string(2) "Z2"
  # php -r '$a="Z9"; $a++; var_dump($a);';
  string(3) "AA0"
  ```
#### php 源码中，字符串自增运算符的算法说明：
  ```C
#define LOWER_CASE 1
#define UPPER_CASE 2
#define NUMERIC 3

static void ZEND_FASTCALL increment_string(zval *str) /* {{{ */
{
	int carry=0;  // 标识是否需要进位
	size_t pos=Z_STRLEN_P(str)-1; // 从字符串末端开始遍历
	char *s;
	zend_string *t;
	int last=0; /* Shut up the compiler warning */
	int ch;

	if (Z_STRLEN_P(str) == 0) {
		zval_ptr_dtor_str(str);
		ZVAL_INTERNED_STR(str, ZSTR_CHAR('1'));
		return;
	}

	if (!Z_REFCOUNTED_P(str)) {
		Z_STR_P(str) = zend_string_init(Z_STRVAL_P(str), Z_STRLEN_P(str), 0);
		Z_TYPE_INFO_P(str) = IS_STRING_EX;
	} else if (Z_REFCOUNT_P(str) > 1) {
		Z_DELREF_P(str);
		Z_STR_P(str) = zend_string_init(Z_STRVAL_P(str), Z_STRLEN_P(str), 0);
	} else {
		zend_string_forget_hash_val(Z_STR_P(str));
	}
	s = Z_STRVAL_P(str);

	do {
		ch = s[pos];
		if (ch >= 'a' && ch <= 'z') {
			if (ch == 'z') { // 当末端是z 时，需要进位,修改为a
				s[pos] = 'a';
				carry=1;
			} else {
				s[pos]++;
				carry=0;
			}
			last=LOWER_CASE;
		} else if (ch >= 'A' && ch <= 'Z') {
			if (ch == 'Z') { // 同理，当末端是Z时，需要进位，修改为A
				s[pos] = 'A';
				carry=1;
			} else {
				s[pos]++;
				carry=0;
			}
			last=UPPER_CASE;
		} else if (ch >= '0' && ch <= '9') {
			if (ch == '9') { // 当末端时9时，需要进位
				s[pos] = '0';
				carry=1;
			} else {
				s[pos]++;
				carry=0;
			}
			last = NUMERIC;
		} else {           
			carry=0;
			break;
		}
		if (carry == 0) {   // 若已经在当前位处理完成，则结束，否则一直处理到第一位
			break;
		}
	} while (pos-- > 0);

	if (carry) {  // 需要进位, 则需要多分配一个byte
		t = zend_string_alloc(Z_STRLEN_P(str)+1, 0);
		memcpy(ZSTR_VAL(t) + 1, Z_STRVAL_P(str), Z_STRLEN_P(str));
		ZSTR_VAL(t)[Z_STRLEN_P(str) + 1] = '\0';
		switch (last) {   //考虑上一位last 标识的是那种类型,赋值不同数据
			case NUMERIC:
				ZSTR_VAL(t)[0] = '1';
				break;
			case UPPER_CASE:
				ZSTR_VAL(t)[0] = 'A';
				break;
			case LOWER_CASE:
				ZSTR_VAL(t)[0] = 'a';
				break;
		}
		zend_string_free(Z_STR_P(str));
		ZVAL_NEW_STR(str, t);
	}
}

  ```
> [php 自增运算符 Doc](https://www.php.net/manual/zh/language.operators.increment.php)
  -----
  路漫漫其修远兮，吾将上下而求索。
