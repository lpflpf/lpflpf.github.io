---
title: "Golang 编码规范"
date: 2023-01-29T10:27:07+08:00
tags:
  - 编码规范
categories:
  - golang

---

[Google Go 语言编码规范（2022-11-23）](https://google.github.io/styleguide/go/decisions)｜[中文版本](https://tonybai.com/google-go-style/google-go-style-decisions/)

>  建议直接[阅读原文](https://google.github.io/styleguide/go/decisions)，以下为精简版本 （10min）

## 命名
- 下划线：仅出现在生成代码的包名中；*_test.go 文件中的Test、Benckmark、Example函数名；cgo或者与操作系统交互的低级库中（例如syscall包中)的可以重复使用的标识符
- 包名：
  - 小写字母，不包含大写、连字符、下划线等；
  - 导入带下划线的包名，需要重命名；
  - 避免使用 util、utility、common、helper 等信息量不足的包
- 方法接收器:
  - 短，1到2个字符，为类型的缩略词，每个receiver 保持一致
- 常量命名
  - 驼峰式，描述值含义
  - eg. MaxPacketSize = 512
- 缩写词
  - 缩写词，同时大写或者小写
  - eg. URL 应写为 urlPony 或者 URLPony，而不是 UrlPony; AppID 而不是 AppId
  - 多个缩写词，则每个缩写词，都需要大写或者小写。eg. xmlAPI XMLAPI等
  - 首字母缩写的，则仅首字母做大小写变换。例如 iOS, gRPC等
- Getters 方法
  - 不使用get前缀，直接用事务名称命名。
  - 涉及复杂远程调用、计算等，使用Fetch、Compute 替代Get；标识需要计算、或阻塞查询
- 变量命名
  - 一般不包含类型名称，除非作用域中有多个相同含义的变量
  - 作用域越大，变量名越长
  - 尽可能保持变量的简洁
  - 单字母变量
    - 方法接收器
    - 常见类型的变量名；r io.Reader, w io.Writer  等
    - 循环变量，循环中变量缩写（小范围）
- 避免重复
  - 包名和导出符号的名称重复，导出符号不应带包的含义；例如 widget.NewWidget → widget.New
  - 变量名中，不出现类型名称；如多种形式出现，使用raw，parsed 等标识。 例如 
  ```golang
  limitRaw := r.FormValue("limit")
  limit, err := strconv.Atoi(limitRaw)
  ```
  - 包名、方法名、类型名、函数名、导入路径、甚至文件名都自动限定了文件中的上下文，避免与所在上下文重复。
## 注释
- 单行注释不宜过长，注意折行
- 所有顶层导出的名字，都需要有注释；不明显的行为或意义的未到处类型或函数也需要注释。
- 注释，首字母大写，始终是完整句子
- 行末注释，可以是简单短语，假定主语是字段名称
- 注释中，尽量提供可以运行的完整例子
- 具名返回值类型
  - 多个相同返回值参数类型，为返回值添加名称
  - 为返回值参数的特定动作，添加注释
  - 不要为了减少函数中的变量声明，而给返回值参数命名
  - 裸返回(return 不带 返回变量）在小函数中可取，中等规模的函数要显示返回  
  - 如果需要在defer 中修改返回值，则需要命名返回值
- 包的注释
  - 在 package 子句上方出现
  - 没有明显主文件，使用doc.go 写包的注释和包声明语句
  - 为维护者准备的注释，放在package 字句后面，不在godoc中显示
## 导入
- 重命名导入包
  - 包含下划线的包名
  - 无有用含义包名，例如 v1
- 分组导入
  - 标准库一个组，重命名一个组，匿名一个组，其他一个组
- 空导入
  - 仅应该在main 包中导入，有助于依赖关系控制
- `import . ；`不建议使用
## error
- 返回错误
  - error 最后一个函数返回参数，标识函数可能失败
  - 不建议仅返回指针，通过是否nil判断是否调用失败
- 错误字符串
  - 不以大写开始，不以标点符号结尾（错误字符串一般打印在上下文中，不是单独出现）
  - 显示完整的错误，一般以大写开头
- 错误处理
  - 一般不会通过 _ 抑制
  - 立即解决，返回调用者，直接`fatal`，`panic`
- 带内错误（in-band 错误，讲错误值与普通返回值混为一谈）
  - 通过返回bool值，判断是否有效。明确错误
- 缩进错误流程
  - 提前处理错误，排除异常情况
  ```golang
  // Good:
  if err != nil {
      // error handling
      return // or continue, etc.
  }
  // normal code
  
  // Bad:
  if err != nil {
      // error handling
  } else {
      // normal code that looks abnormal due to indentation
  }
  ```
- if 中初始化的变量，若多处使用，则将变量移出

## 语言
- 字段名
  - 外部包的类型，字段名复制，需要指定字段名
  - 小型、简单的结构体，可省略字段名
- 括号匹配
  - 一对大括号，收尾括号应该出现在缩进量与开头大括号相同的一行中
  - 多行结构字面值，首位括号应出现在下一行
  - 拥抱式大括号。？？
- 重复的类型名，省略，增加可读性
```golang
// Good:
good := []*Type{
    {A: 42},
    {A: 43},
}
// Bad:
repetitive := []*Type{
    &Type{A: 42},
    &Type{A: 43},
}
```
- 零值字段
  - 省略零值字段的复制，提高可读性
- 空切片，最好使用 var t []string 声明；
  - 不强迫用户区分数组的nil 与空切片（可以使用返回error的形式）
  - 入参是切片，通过判断切片长度，而不是nil
- 避免缩进混乱
- 函数格式化
  - 保持函数、方法签名在同一行，参数过多，缩短函数参数（参考最佳实践）。
  - 调用时，也不使用多行。通过传入结构体，或者函数文档描述参数细节；修改不了api，也可以通过换行实现（按照语义分组换行）
- 条件与循环
  - if语句、for 语句、switch、case，不应该换行
  - 可以通过提取bool操作；提取局部变量值，缩短代码
  - case 过长，索引所有case，避免混乱
  - 避免尤达表达式；变量、常数判等，变量值放在判等运算符左侧
- 复制
  - （一般情况）T类型的值，T成员包含指针，一般不能复制
- 不要panic
  - 正常的错误处理，不要panic
  - main包，初始化代码中，使用 log.Exit 处理终止程序的错误（不会运行defer 方法）
  - 失败要panic的方法，一般以Must开头
- goroutine 生命周期
  - 创建时，明确是否退出，何时退出
  - 将同步的代码，限制在一个同步函数内
- 接口
  - 定义在使用接口的包中，而不是实现接口类型的包中。实现包返回具体的类型。
  - 如果包的用户不需要为他们传递不同类型的参数，则不使用接口类型参数
- 泛型
  - 如果有几个类型共享一个有用的统一接口，可以考虑抽象为接口，可能不需要泛型
  - 否则，与其依赖任何类型和过度的类型转换，可以考虑泛型
- 传值
  - 不要为了节省几个字节，而把指针作为传参。eg. string, 接口指针
  - 大型结构体，一般使用指针传递参数。
- 接收器
  - 修改，需要用指针，否则不需要
  - 包含不能安全复制的字段，需要用指针
  - 内置类型，不需要修改，使用值类型
  - 接收器是 map, func, channel, 使用值而不是指针
  - 接收器是小数组，结构体，元素没有可变字段和指针的值类型，使用值类型
  - 不确定：使用指针类型
  - 要么都是指针方法，要么都是值方法
- switch/break
  - switch 子句末尾，不添加多余 break
  - 空case，请使用注释
- 同步函数
  - 单独goroutine中调用，增加并发
- 类型别名
  - 用于迁移源码位置，尽量不用
- 使用 %q，而不是手动加引号（可以很明显看出空字符串）
- any，新的代码倾向于使用any，而不是interface{}
## 常用库
- Flags
  - 标识名使用蛇形命名，变量名驼峰命名
  - 传入参数在main或者等价包中定义，不引入
- 日志包
  - 异常退出，使用 log.Fatal 退出（包含堆栈），log.Exit退出（不包含堆栈）
- contexts
  - 传递到一个函数或方法是，context 是第一个参数
  - 例外：
    - http处理请求 , req.Context()
    - rpc 方法，来自于流
  - 结构体类型，不要添加上下文成员
- 自定义上下文
  - 不要创建自定义上下文，没有例外。
- crypto/rand
  - 不加种子，生成器完全可预测；使用nanoseconds做种子，只有几个比特的熵；
  - 使用 rand.Reader
## 有用的测试失败
- 识别函数，失败信息，需要包含函数本身
- 识别输入，短函数，应该包含输入；否则，测试用例名称中增加case描述信息
- got 在want之前，使用got，和want标识
- 结构体整体比较，使用深度比较法，或github.com/google/go-cmp （复杂比较，go团队维护，为测试而生，生产不适合使用）比较
- 比较稳定的结果，比较结构体，而不是编码结果
- 持续进行，失败后，继续测试，将所有问题暴露
- 相等性比较和差异（cmp库）
- 详细程度 （大多是 "YouFunc(%v) = %v, want %v" 足够）
- 打印差异
- 测试错误语义，不实用字符串比较。因为可变性很大，将单元测试变成了变化检查器

## 测试的结构组织

- 子测试，命名：能使用Bazel测试过滤器标记运行，好区分过滤
- 表驱动的测试，多个测试case放切片，批量测试；不通逻辑的检查，使用多个测试函数
- 数据驱动的测试用例，将测试用例区分，而不是放在一个测试case中
- 测试助手，执行设置和清理任务的函数。测试助手中的错误，被认为是环境故障
- 同一包下的测试，可访问未导出表示，可提升覆盖率和间接性
- 不同软件包的测试；在同一个文件夹下，两个包名，防止冲突
- testing包，唯一被允许使用的测试框架


## 非决定性（未达成共识的case）


## 其他：

- [uber 语言编码规范](https://github.com/xxjwxc/uber_go_guide_cn)
- [go静态检查工具 staticcheck](https://github.com/dominikh/go-tools)
- [go clean code](https://github.com/Pungyeon/clean-go-article)

