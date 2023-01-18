---
title: golang database 中 Scanner/ Valuer 接口学习
date: 2021-01-11 11:47:35
tags:
  - golang
  - database/sql
category:
  - golang

---

自定义类型在做数据库查询和插入操作时，可以通过实现 Scanner / Valuer 接口，使我们在做db的增删查改时更加顺畅。


<!--more-->

### 现实中遇到的问题

在做后台系统时，有些表中的字段是定制化的，例如:

```go
type Day time.Time  // 以天为单位标记
type LocaleTime  time.Time // 本地格式化的时间, 存在 0000-00-00 00:00:00 值
```

为什么需要给这些类型重定义? 

我这里的原因是为了在给下游提供JSON接口时输出标准化的值。例如，对于Day 类型，需要输出 "2021-01-11"，对于 LocaleTime 需要输出的是 "2021-01-11 12:41:01"。

对于JSON 的格式化，使用的是如下接口：

```go
const dayFormat = "2006-01-02"

// 格式化JSON 解析
func (t *Day) UnmarshalJSON(data []byte) (err error) {
    now, err := time.ParseInLocation(`"`+dayFormat+`"`, string(data), time.Local)
    *t = Day(now)
    return
}

// 格式化JSON 编码
func (t Day) MarshalJSON() ([]byte, error) {
    b := make([]byte, 0, len(dayFormat)+2)
    b = append(b, '"')
    b = time.Time(t).AppendFormat(b, dayFormat)
    b = append(b, '"')
    return b, nil
}

const localTimeFormat = "2006-01-02 15:04:05"


func (t *LocalTime) UnmarshalJSON(data []byte) (err error) {
    now, err := time.ParseInLocation(`"`+localTimeFormat+`"`, string(data), time.Local)
    *t = LocalTime(now)
    return
}

func (t LocalTime) MarshalJSON() ([]byte, error) {
    b := make([]byte, 0, len(localTimeFormat)+2)
    b = append(b, '"')
    b = append(b, []byte(t.String())...)
    //b = time.Time(t).AppendFormat(b, localTimeFormat)
    b = append(b, '"')
    return b, nil
}

func (t LocalTime) String() string {
    if time.Time(t).IsZero() {
        return "0000-00-00 00:00:00"
    }

    return time.Time(t).Format(localTimeFormat)
}
```

解决了JSON 编解码的问题，但是对于DB插入和查询却总是有问题。

### 如何解决

经过翻看接口文档，其实解决方式和 `json.UnmarshalJSON` / `json.MarshalJSON` 接口类似。只要实现该类型的Valuer/Scanner接口即可。

```go
func (t Day) Value() (driver.Value, error) {
    tTime := time.Time(t)
    return tTime.Format("2006/01/02 15:04:05"), nil
}

func (t *Day) Scan(v interface{}) error {
    switch vt := v.(type) {
    case time.Time:
        *t = Day(vt)
    case string:
        tTime, _ := time.Parse("2006/01/02 15:04:05", vt)
        *t = Day(tTime)
    }
    return nil
}

func (t LocalTime) Value() (driver.Value, error) {
    if time.Time(t).IsZero() {
        return "0000-00-00 00:00:00", nil
    }
    return time.Time(t), nil
}

func (t *LocalTime) Scan(v interface{}) error {
    switch vt := v.(type) {
    case time.Time:
        *t = LocalTime(vt)
    case string:
        tTime, _ := time.Parse("2006/01/02 15:04:05", vt)
        *t = LocalTime(tTime)
    default:
        return nil
    }
    return nil
}
```

### 接口学习

下面，具体学习下两个接口。

#### Scanner 接口

```go
type Value interface{}
type Valuer interface {
    // Value returns a driver Value.
    Value() (Value, error)
}

func IsValue(v interface{}) bool {
    if v == nil {
        return true
    }
    switch v.(type) {
    case []byte, bool, float64, int64, string, time.Time:
        return true
    }
    return false
}
```

在sql的Exec 和 Query 时，需要将入参转换为各db驱动包支持的数据类型。而Valuer 是将值转换为 driver.Value 类型的接口定义。

因此，对于自定义的时间转义，可以转义为一个 time.Time 类型，或者一个字符串类型。

#### Valuer 接口

```go
type Scanner interface {
    Scan(src interface{}) error
}
```

在数据查询结果中，需要将查询结果映射为go支持的数据类型。在实现时，首先会把所有的数据都转换为 int64, float64, bool, []byte, string, time.Time, nil 几种类型，然后调用目标类型的Scan方法赋值。实现Scanner 接口时，入参即为这些类型中的一种，仅需把入参转为我们的变量即可。


