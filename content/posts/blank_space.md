---
title: 你不知道的空格
date: 2020-04-08 15:23:04
tags:
---

本文对了解的空格分为几个Level，看大家能达到哪个level。

<!--more-->

### Level1： 半角空格

历史最悠久的空格，在1967年，ASCII 规范中被定义。
空格在 ASCII 中编码为0x20, 占位符为一个半角字符。在日常英文书写和代码编写中使用。

### Level2: 全角空格

中文输入中的空格（标准说法为中日韩表意字符(CJK)中使用的宽空格）。和其他汉字一样，作为GBK的一个字符，其对应的unicode码为\u3000.宽度是2个半角空格的大小。
例如：
```
  国父　孙中山先生 
```

### Level3: 不间断空格 ( non-breaking space )

unicode 为 \u00A0, 在代码中可能会出现的编码错误(utf8 编码0xC2 0xA0) 就是它了。
在Word中，会遇到一个有多个单词组成的词组被分割在两行文字中，这样很容易让人看不明白。这时候，不间断空格就可以上场了。
输入不间断空格，会将不间断空格连着的单词在一行展示。
举个例子：

![](no-breaking-space.jpg)

上面英文使用了不间断空格，下面没有使用。所以上面的英文自动在一行展示，而下面没有。
在word中输入不间断空格的方式为: (Ctrl + Shift + Space)

除了在word等文本编辑软件中使用，其实不间断空格在html 中大量使用。\&nbsp; 是html 中最为常见的空格。由于html页面中，如果有多个连着的半角空格，则空格只会展示一个。而使用\&nbsp; 空格，则会显示占位半个自宽。

### Level4: 零宽度空格 (ZERO WIDTH SPACE)

零宽度空格有两种

1. 零宽度空格 unicode 编码为 \u200B.  

不可见非打印字符。有了半角空格，也有了全角空格，其实还有零宽度空格。因为宽度为零，因此该字符是一个不可见字符。
这个编码虽然是不可见的，但是也是非常有用的。它可以替换html中的<wbr/>标签(软换行, html5 新增)。

2. 零宽度非中断空格(ZWNBSP) unicode 编码为 \u2060  (之前使用\ufeff表示，unicode 3.2 开始 \ufeff 标记unicode文档的字节序。)
    该空格结合了 non-breaking space 和 零宽度空格的特点。既会自动换行，宽度又是0。

零宽度空格（软换行）举例：

一行连续的英文编码:
```
<p style="font-size:100px;">PhpIsTheBestProgramingLanguageInTheWorld</p>
```
chrome 中将显示不换行：

![](zero_width_space1.jpg)
而如果在每个可以换行的地方加上 \<wbr /\>, 则可以在标记的最近的地方换行。
```
<p style="font-size:100px;">Php<wbr />Is<wbr />The<wbr />Best<wbr />Programing<wbr />Language<wbr />In<wbr />The<wbr />World</p>
```
chrome 中将显示：

![](zero_width_space2.jpg)

### Level5: 其他空格字符空格

虽然已经有半角空格、全角空格，但是上面的空格如果字体变化了，不会随着字体的变化而变化。
因此，又有了可以随着字体的变化而变化的空格，简单罗列如下：

在html 的宽度度量中，有一种单位叫em，是按照字体大小定义的，下面的em也是字体的宽度。
打印字符的空格有很多种，罗列几个：

  | 名称 | unicode 编码 | html 标记 | 特征和用途 |
  |: - :|: - :|: - :|: - :|
  | 短空格     |  \u2002   |  \&ensp; | html 中占位半个字 |
  | 长空格     |  \u2003   | \&emsp; | html 中占位一个字 |
  | 1/3em空格  | \u2004 | \&emsp13; | 占用1/3个空格 |
  | 1/4em空格  | \u2005 | \&emsp14;  | 占用1/4个空格 |
  | 1/6em空格  | \u2006 | \&emsp14; |  占用1/6个空格 |
  | 数样间距 (figure space) | \u2007 | \&numsp; |  在等宽字体中，宽度是一个字符的宽度。|
  | 行首前导空格 (punctuation space)|  \u2008 | \&puncsp; | 宽度约为 0x20 的宽度。 |
  | 瘦弱空格 (thin space) | \u2009 | \&thinsp; | 宽度是 全角打印空格的 1/5 或者 1/6 (宽度不定,法文设置为1/8)， 主要用在打印两个空的引号之间。|
  | hair space| \u200a |  \&hairsp; | (浏览器目前不支持), 最窄的空格，推荐标准为 (1/10, 1/16) |
  | narrow no-break space |\u202f |\&nnbsp;|和0a 类似，不同语种中不太一样。|
  | medium mathematical space |\u205f | \&mediumspace; |  在格式化数学公式时使用。是 4/18 的 em宽度，例如："a + b"中，a 和+ 之间应该用 这个空格|

![](/images/weixin_logo.png)
  
-----
引用链接:

- https://en.wikipedia.org/wiki/Zero-width_space
- https://en.wikipedia.org/wiki/Em_(typography)
- https://en.wikipedia.org/wiki/En_(typography)
- https://en.wikipedia.org/wiki/Zero-width_space
- https://en.wikipedia.org/wiki/Thin_space
- https://en.wikipedia.org/wiki/Whitespace_character
- https://www.unicode.org/charts/PDF/U2000.pdf
- https://web.archive.org/web/20100314135826/https://www.microsoft.com/typography/developers/fdsspec/spaces.htm
- https://en.wikipedia.org/wiki/Word_joiner
