---
title: golang DB query 的优化之旅
date: 2021-09-30 13:57:59
tags:
  - golang
  - database query
---

记录一次 Golang 数据库查询组件的优化。
<!--more-->

# 背景介绍

线上有一块业务，需要做大量的数据库查询以及编码落盘的任务。数据库查询20分钟左右，大约有2kw条sql被执行。如果可以优化数据库查询的方法，可以节省一笔很大的开销。

由于代码比较久远，未能考证当时的数据查询选型为什么不适用orm，而是使用原生的方式自己构建。下面是核心的数据查询代码：

``` golang
func QueryHelperOne(db *sql.DB, result interface{}, query string, args ...interface{}) (err error) {

    // 数据库查询
	var rows *sql.Rows
	log.Debug(query, args)
	rows, err = db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	// 获取列名称，并转换首字母大写，用于和struct Field 匹配
	var columns []string
	columns, err = rows.Columns()
	if err != nil {
		return err
	}

	fields := make([]string, len(columns))
	for i, columnName := range columns {
		fields[i] = server.firstCharToUpper(columnName)
	}

    // 传参必须是数组 slice 指针
	rv := reflect.ValueOf(result) 
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	} else {
		return errors.New("Parameter result must be a slice pointer")
	}
	if rv.Kind() == reflect.Slice {
		elemType := rv.Type().Elem()
		if elemType.Kind() == reflect.Struct { 
			ev := reflect.New(elemType)                
            // 申请slice 数据，之后赋值给result
			nv := reflect.MakeSlice(rv.Type(), 0, 0)  
			ignoreData := make([][]byte, len(columns))

			for rows.Next() { // for each rows
                // scanArgs 是扫描每行数据的参数
                // scanArgs 中存储的是 struct 中field 的指针
				scanArgs := make([]interface{}, len(fields))
				for i, fieldName := range fields {
					fv := ev.Elem().FieldByName(fieldName)
					if fv.Kind() != reflect.Invalid {
						scanArgs[i] = fv.Addr().Interface()
					} else {
						ignoreData[i] = []byte{}
						scanArgs[i] = &ignoreData[i]
					}
				}
				err = rows.Scan(scanArgs...)
				if err != nil {
					return err
				}
				nv = reflect.Append(nv, ev.Elem())
			}
			rv.Set(nv) 
		}
	} else {
		return errors.New("Parameter result must be a slice pointer")
	}

	return
}
```

方法通过如下方式调用：

```golang
type TblUser struct {
    Id          int64
    Name        string
    Addr        string
    UpdateTime  string
}

result := []TblUser{}
QueryHelperOne(db, &result, query, 10)
```
# 逐步优化

直接看上面的代码，发现没有什么大的问题，但是从细节上不断调优，可以让性能压榨到极致。

## 网络优化

golang 提供的db.Query(sql, args...) 方法，内部的实现，也是基于prepare 方法实现的。
prepare 有三个好处：

    - 可以让 mysql 省去每次语法分析的过程
    - 可以避免出现sql 注入
    - 可以重复使用prepare 的结果，只发送参数即可做查询

但是，也有不好的地方。一次 db.Query 会有三次网络请求。

   -  prepare 
   -  execute
   -  closing

而如果有多次相同SQL 查询的话，这种方式是非常占优的。因此，可以使用prepare 替换 db.Query 减少一次网络消耗。

```golang

var stmts = sync.Map{}
func QueryHelperOne(db *sql.DB, result interface{}, query string, args ...interface{}) (err error) {

    // 使用sync.Map 缓存 query 对应的stmt
    // 减少不必要的prepare 请求
    var stmt *sql.Stmt
    if v, ok := stmts.Load(query); ok {
        stmt = v.(*sql.Stmt)
    } else {
        if stmt, err = db.Prepare(query); err != nil {
            return err
        } else {
            stmts.Store(query, stmt)
        }
    }

    var rows *sql.Rows
    log.Debug(query, args)
    rows, err = stmt.Query(args...)
    if err != nil {
        _ = stmt.Close()
        stmts.Delete(query)
        return err
    }
    defer rows.Close()

    // 后面代码省略 ...
}
```

通过此番修改，作业的性能提升了17%，效果还是非常明显的。

## gc 优化

### 优化1
在服务中，会预申请slice空间，因此无需每次构建的时候重新申请slice 内存。

```golang

// old code
// nv := reflect.MakeSlice(rv.Type(), 0, 0)  
// new code
nv := rv.Slice(0, 0)
```

### 优化2
从代码56 行可以看到，每次会append 数据到数组中。由于 结构体切片在append 时，是做内存拷贝；scanArgs 的数据由于每次scan 都会覆盖，因此可以复用，不需要每次rows 的时候映射。

```golang
ev := reflect.New(elemType)                
// 申请slice 数据，之后赋值给result
nv := reflect.MakeSlice(rv.Type(), 0, 0)  
ignoreData := make([][]byte, len(columns))
// scanArgs 是扫描每行数据的参数
// scanArgs 中存储的是 struct 中field 的指针
scanArgs := make([]interface{}, len(fields))
for i, fieldName := range fields {
	fv := ev.Elem().FieldByName(fieldName)
	if fv.Kind() != reflect.Invalid {
		scanArgs[i] = fv.Addr().Interface()
	} else {
		ignoreData[i] = []byte{}
		scanArgs[i] = &ignoreData[i]
	}
}
for rows.Next() { // for each rows
    err = rows.Scan(scanArgs...)
	if err != nil {
		return err
	}
	nv = reflect.Append(nv, ev.Elem())
}
rv.Set(nv) 

```

减少了每行扫描的时候，新申请scanArgs

### 优化 3

对于不在field中的数据，需要使用一个空的值代替，上面代码使用的是一个[]byte 的切片，其实只需要一个[]byte 即可。代码如下：

```golang
ignoreData := []byte{}
// scanArgs 是扫描每行数据的参数
// scanArgs 中存储的是 struct 中field 的指针
scanArgs := make([]interface{}, len(fields))
for i, fieldName := range fields {
	fv := ev.Elem().FieldByName(fieldName)
	if fv.Kind() != reflect.Invalid {
		scanArgs[i] = fv.Addr().Interface()
	} else {
		scanArgs[i] = &ignoreData
	}
}
```

### 优化 4

由于相同的sql会查询次数在千万级；因此可以把每次扫描行所需要的行元素ev,以及对应的扫描参数列表 scanArgs 都缓存起来，再使用时从内存中加载即可。

```golang
// 定义数据池，用于存储每个sql 对应的扫描行item 以及扫描参数
// 全局代码
var datapools = sync.Map{}

type ReflectItem struct {
    Item     reflect.Value
    scanArgs []interface{}
}


///////// 方法调用内部

// 从数据池中加载query 对应的 ReflectItem
if v, ok := datapools.Load(query); ok {
    pool = v.(*sync.Pool)
} else {
    // 构建reflectItem
	var columns []string
	columns, err = rows.Columns()
	if err != nil {
		return err
	}

    pool = &sync.Pool{
        New: func() interface{} {
            fields := make([]string, len(columns))
            for i, columnName := range columns {
                fields[i] = server.firstCharToUpper(columnName)
            }

            ev := reflect.New(elemType) // New slice struct element
            // nv := reflect.MakeSlice(rv.Type(), 0, 0) // New slice for fill
            ignored := []byte{}
            scanArgs := make([]interface{}, len(fields))
            for i, fieldName := range fields {
                fv := ev.Elem().FieldByName(fieldName)
                if fv.Kind() != reflect.Invalid {
                    scanArgs[i] = fv.Addr().Interface()
                } else {
                    scanArgs[i] = &ignored
                }
            }
            return ReflectItem{
                Item:     ev,
                scanArgs: scanArgs,
            }
        },
    }
    datapools.Store(query, pool)
}
ri = pool.Get().(ReflectItem)

// 复用 ev 和 scanArgs
ev = ri.Item
scanArgs = ri.scanArgs

// 开始扫描
nv := rv.Slice(0, 0)
for rows.Next() { // for each rows
    err = rows.Scan(scanArgs...)
    if err != nil {
        return err
    }
    nv = reflect.Append(nv, ev.Elem())
}
rv.Set(nv) // return rows data back to caller
pool.Put(ri)
// 结束扫描
```

经过几次优化，24分钟执行完的作业，成功减少到了18分钟。

# 总结

- golang prepare 的实现，需要进一步了解，在使用prepare的情况下，连接是如何复用的，比较困惑。
- 对于相同query 的情况，但是扫描struct 类型不同的情况，会有问题。扫描参数的数据池，应该使用结构体类型做key。
