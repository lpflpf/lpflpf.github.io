---
layout:     post
title:      "mysql hint 学习"
subtitle:   "hint mysql"
date:       2018-05-26
author:     "李朋飞"
header-img: "img/post-bg-2015.jpg"
category:   mysql
tags:
    - Mysql
    - Hint
---

### Index Hint
- [官网说明](https://dev.mysql.com/doc/refman/5.6/en/index-hints.html)
- 语法说明

```
tbl_name [[AS] alias] [index_hint_list]

index_hint_list:
    index_hint [index_hint] ...

    index_hint:
        USE {
            INDEX|KEY}
                  [FOR {
                      JOIN|ORDER BY|GROUP BY}] ([index_list])
      | IGNORE {
          INDEX|KEY}
                [FOR {
                    JOIN|ORDER BY|GROUP BY}] (index_list)
      | FORCE {
          INDEX|KEY}
                [FOR {
                    JOIN|ORDER BY|GROUP BY}] (index_list)

    index_list:
        index_name [, index_name] ...
```


在表明后 使用 (USE \| IGNORE \| FORCE ) (INDEX\|KEY)`[FOR (JOIN \| ORDER BY \| GROUP BY)]  index_name ...

-  **使用的是索引名称， 而非列名称**
-  对于自然语言模式下的全文搜索, index hints默认不起作用，单索引仍有效。
-  对于boolean模式下的全文搜索，index hints 对于 For Order \| For Group 模式，默认屏蔽。对于 for join 或者没有for的情况生效。

###  Query Cache SELECT Options

- [官网说明](https://dev.mysql.com/doc/refman/5.6/en/query-cache-in-select.html)
- 语法说明：

```
    SELECT SQL_CACHE id, name FROM customer;
    SELECT SQL_NO_CACHE id, name FROM customer;
```

- **指定SQL 是否需要在缓存中查找结果。对于一天执行1，2次的SQL，可以使用 SQL_NO_CACHE 使其不从缓存查找**

### LOW_PRIORITY \| HIGH_PRIORITY

- [官方说明](https://dev.mysql.com/doc/refman/5.6/en/select.html)
- [官方说明](https://dev.mysql.com/doc/refman/5.6/en/insert.html)

- 在INSERT \| SELECT 语句中说明。
- 在INSERT 中使用LOW_PRIORITY,  则该语句将在没有客户端读该表的时候执行。（在读频繁的表中，这个会引发饥饿现象）
- INSERT HIGH_PRIORITY(默认情况),除非使用了该配置:--low-priority-updates 
- SELECT LOW_PRIORITY (默认情况),
- SELECT HIGH_PRIORITY. 只有查询一次，并且需要非常快的情况下才会使用该HINT. 这种情况会所表，知道查询结束.

### INSERT DELAYED

- [官方说明](https://dev.mysql.com/doc/refman/5.6/en/insert-delayed.html)

- 在特定的存储引擎中可以使用。(MyISAM, MEMORY, ARCHIVE, and BLACKHOLE tables), **Innodb 并不支持**.
- 延时插入，异步返回。将插入的任务加入到队列中。
