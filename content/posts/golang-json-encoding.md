---
title: json map 和 struct 编码对比
date: 2019-06-27 13:55:40
tags:
  - golang
category: golang
---

本文对比试验采用官方包做json map 和struct 编码。 

<!--more-->


> `encoding/json`

#### 数据构造
map 数据类型为map[string]string , key 长度为10, val 长度为100
struct 定义如下：
``` golang
type Object struct {
	Xvlbzgbaic string `json:"xvlbzgbaic"`
	Krbemfdzdc string `json:"krbemfdzdc"`
	Rzlntxyeuc string `json:"rzlntxyeuc"`
	Ctzkjkziva string `json:"ctzkjkziva"`
	Orsufumaps string `json:"orsufumaps"`
	Hyevwbtcml string `json:"hyevwbtcml"`
	Baatlyhdao string `json:"baatlyhdao"`
	Fkfohsvvxs string `json:"fkfohsvvxs"`
	Pqwarpxptp string `json:"pqwarpxptp"`
	Orvaukawww string `json:"orvaukawww"`
}
```

对比程序如下：

```go
	obj := Object{}
	json.Unmarshal([]byte(str), &obj)

	start := time.Now()
	for i := 0; i < 1000000; i++ {
		json.Marshal(obj)
	}

	fmt.Println(time.Since(start))

	maps := map[string]string{}
	json.Unmarshal([]byte(str), &maps)

	start = time.Now()
	for i := 0; i < 1000000; i++ {
		json.Marshal(maps)
	}
	// 
	fmt.Println(time.Since(start))
```

其中，str 为生成好的固定json数据, 我们对相同的数据做json 编码, 运行结果可以看出，时间差距大约为1倍，若将map的key 个数调整为100个

运行次数均为1000,000 次

| type\\ keys 个数 | 10 | 100 | 1000 | 
|:----------------:|:---|:----|:-----|
| struct |3.84s | 33.72s | 5m42.34s |
| map[string]string | 7.59s| 1m20.03s | 17m21.47s  |
| no sorting map[string]string | 6.40s | 57.61s | 10m4.39s |


>从上述对比中，得出如下结论：  
>**在大量使用json 编码时(尤其是map结构较大时)，请注意尽量直接用struct，而不是用map做编码。**

#### 原因探究

- map 编码问题
  - struct 多次压缩时，encoding 中会缓存 name 信息, 以及对应val的类型，直接调用相应的encoder 即可;相反，map 则每次需要对key 做反射,根据类型判断获取key的值，val值也需要反射获取相应的encoder，时间浪费较多。
  - map 在做json 的解析的结果，会做排序操作。若修改源码，将排序操作屏蔽,key 越多，需要的时间越多。

- map 编码

```go
// go/src/encoding/json/encode.go 
func (me *mapEncoder) encode(e *encodeState, v reflect.Value, opts encOpts) {
	if v.IsNil() {
		e.WriteString("null")
		return
	}
	e.WriteByte('{')

	// Extract and sort the keys.
	keys := v.MapKeys()
	sv := make([]reflectWithString, len(keys))
	for i, v := range keys {
		sv[i].v = v
		if err := sv[i].resolve(); err != nil {
			e.error(&MarshalerError{v.Type(), err})
		}
	}
    // 在输出前会做key 的排序，最后按照key 排序的结果做输出
	sort.Slice(sv, func(i, j int) bool { return sv[i].s < sv[j].s })

	for i, kv := range sv {
		if i > 0 {
			e.WriteByte(',')
		}
		e.string(kv.s, opts.escapeHTML)
		e.WriteByte(':')
		me.elemEnc(e, v.MapIndex(kv.v), opts)
	}
	e.WriteByte('}')
}
```

- struct 编码

```go
// go/src/encoding/json/encode.go


type structEncoder struct {
	fields    []field
	fieldEncs []encoderFunc
}

func newStructEncoder(t reflect.Type) encoderFunc {
	fields := cachedTypeFields(t) // 从cache 中获取fields
	se := &structEncoder{
		fields:    fields,
		fieldEncs: make([]encoderFunc, len(fields)),
	}
	for i, f := range fields {
		se.fieldEncs[i] = typeEncoder(typeByIndex(t, f.index))
	}
	return se.encode
}

func (se *structEncoder) encode(e *encodeState, v reflect.Value, opts encOpts) {
	e.WriteByte('{')
	first := true
	for i, f := range se.fields {   // fields 被缓存在structEncoder 结构体中
		fv := fieldByIndex(v, f.index)
		if !fv.IsValid() || f.omitEmpty && isEmptyValue(fv) {
			continue
		}
		if first {
			first = false
		} else {
			e.WriteByte(',')
		}
		e.string(f.name, opts.escapeHTML)
		e.WriteByte(':')
		opts.quoted = f.quoted
		se.fieldEncs[i](e, fv, opts)
	}
	e.WriteByte('}')
}


```

#### json-iterator/go

根据上述内容，对比github.com/json-iterator/go 与 encoding/json 的对比试验，也可以看出，iterator 对 map 的性能提升不是很明显(由于都需要做反射)，后续将做试验验证。

-----
#### Env
- **机器环境**： 1C1G
- **golang 版本**: go1.10.3 linux/amd64
