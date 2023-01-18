---
title: Python 操作 Excel 和 Word
date: 2020-04-02 11:35:00
tags:
  - Python
  - Excel
  - Word
  - win32com

---

有重复的工作，那就尽量让这些搬砖的事情用程序来替代，省下的时间多陪陪家人。

<!--more-->

> Python 作为一门解释型语言，又是一种动态类型的语言，其灵活性非常适合编写日常脚本。
> 一些日常不注重效率的需求可以用 `Python` 来实现。何况`Python`有足够的开源依赖包供我们使用。
> 本文主要介绍通过 `Python` 语言实现对 Excel 和 Word 的操作，以及可能出现的坑。

# 几种选择

  Python 对 Excel，Word 的操作选择其实不是很多。主要分类两类。

  - [Win32Com](https://github.com/mhammond/pywin32) 通过调用Win32Api实现操作Word, Excel, PowerPoint
  - [python-docx](https://github.com/python-openxml/python-docx)(word 读写）, [python-excel](http://www.python-excel.org/) (excel 操作），在Windo32Api 之上实现对Word，Excel 的操作, 文档比较齐全和丰富

## Win32Com

Windows 对 Excel, Word, PowerPoint 等应用程序会提供专门的Com包供开发者使用。Win32Com 是对包的简单封装，接口层基本无变化。
在很多博客中头疼对Win32Com 的接口不是很了解，无法开发。其实Windows 官方提供了详细的接口文档，和 VBA 语言接口是基本一致的。
因此可以参考如下文档的实现：

  - [Excel](https://docs.microsoft.com/zh-cn/office/vba/api/overview/excel/object-model)
  - [Word](https://docs.microsoft.com/zh-cn/office/vba/api/overview/Word/object-model)
  - [PowerPoint](https://docs.microsoft.com/zh-cn/office/vba/api/overview/powerpoint/object-model)

举个例子：

在Excel 中，Sheet 有两种形式，Charts 或者 Worksheet, 下面举例从 Chart Sheet 中获取图片，并导出到本地。

```python
import win32com.client as win32

# 从Excel excel_name 的sheet_name 中导出图片保存至picture_name 中
def export_picture(excel_name, picture_name, sheet_name):
    # 获取Excel api
    excel = win32.gencache.EnsureDispatch("Excel.Application")
    
    # 打开Excel 文档, wb 为文件句柄
    wb = excel.Workbooks.Open("excel_name.xlsx")
    
    # 导出图片
    wb.Sheets("sheet_name").Export("picture_name.jpg")
    
    wb.Close()

```

## python-docx 包

使用win32com 操作会有一些不方便，可以使用docx 库。 docx 库使用比较人性化。

doc 是按照回车符分割为一个一个段落、heading 等。因此如果需要插入一个回车符，那就需要插入一个paragraph。

举个例子：

```python

import docx

def edit_doc(doc_name, text):
    doc = docx.Document(doc_name)

    # 添加文字，并居中
    # 此处可直接添加文字，add_paragraph 默认会调用 add_run
    doc.add_paragraph(text).paragraph_format.aligenment = docx.enum.text.WD_ALIGN_PARAGRAPH.CENTER

    # 添加空段落
    doc.add_paragraph()


    # 添加 10x10 的表格
    rows = 10
    cols = 10
    table = doc.add_table( rows, cols, style="Table Grid")

    # 对表格内容进行赋值
    for x,y in [(x,y) for x in range(0,9) for y in range (0, 9)]:
        table.cell(x,y).text = str(x * y)
        table.cell(x,y).paragraphs[0].paragraph_format.alignment = docx.enum.text.WD_ALIGN_PARAGRAPH.CENTER
        # 设置 cell 的宽度
        table.cell(x,y).width = 25600 * 30

    # 第一行表格的合并
    table.cell(0,0).merge(table.cell(0,9))

    # 插入分页符
    doc.add_page_break()

    # 插入一张图片
    # 对word 的编辑，需要通过 add_run() 来实现
    doc.add_paragraph().add_run().add_picture("pciture_name.jpg", 25600 * 200, 25600 * 200)

    # 保存文件
    doc.save("output.docx")
    
```


## python-excel 的使用

excel 的操作，有两个包, xlrd 用于excel 的读取， xlwt 是用于excel 的写操作。这里只对excel 的读取简单介绍。

``` python

# 获取表格的数据 [0, rows] x [0, cols]
def get_table_data(excel_name, sheet_name, rows, cols):
    result = []

    book = xlrd.open_workbook(excel_name)
    sheet = book.sheet_by_name(sheet_name)
    for row in range (0, rows):
        line = []
        for col in range (0, cols):
            line.append(sheet.cell_value(row, col))
        result.append(line)

    return result
```

遇到的一个问题:
如何判断表格内容为空: 

```python

if sheet.cell_type(x,y) is 0:
    print "is empty"

```


# 技术总结

> 操作Word，Excel 的包还是比较丰富的，以上是使用比较多的几个。
> 对于在xlrd, xlwt, docx 中没有实现的接口，可以使用win32com 来实现。
> 如果win32com 无法实现，则可以考虑是否应用程序没有提供相应的接口服务了。
