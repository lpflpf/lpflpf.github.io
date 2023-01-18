---
title: pika hash 表 知识总结
date: 2021-01-29 13:32:22
author: 李朋飞
tags:
  - pika
  - cache
---

本文主要对 pika 中 Hash 数据结构的使用做一个小结。

<!--more-->

> [Pika](https://github.com/Qihoo360/pika) 是 360 开源的一个非关系型数据库，可以兼容 Redis 系统的大部分命令。支持主从同步。主要区别是 Pika 支持的数据量不受内存的限制，仅和硬盘大小有关。底层使用了 RocksDB 做 KV 数据的存储。

本文主要对Pika 的 Hash 数据结构做个小结。

## 命令支持

| 接口          | 状态  |
|  :-:          |  :-:  |
| HDEL          | 支持  | 
| HEXISTS       | 支持  |
| HGET          | 支持  |
| HGETALL       | 支持  |
| HINCRBY       | 支持  |
| HINCRBYFLOAT  | 支持  |
| HKEYS         | 支持  |
| HLEN          | 支持  |
| HMGET         | 支持  |
| HMSET         | 支持  |
| HSET          | 暂不支持单条命令设置多个field value，如有需求请用HMSET |
| HSETNX        | 支持  |
| HVALS         | 支持  |
| HSCAN         | 支持  |
| HSTRLEN       | 支持  |

## 存储引擎

由于 Pika 数据最终会进入RocksDB,而RocksDB仅支持K-V数据结构, 因此 需要把两层结构的 Hash 数据转换为一层的KV存储结构。例如，执行如下的命令：

```
HSET key field value
```

Pika首先将创建 hash 的 meta k-v 值，用来保存 hash 结构的元数据, 其数据格式如下：

![](pika-hash-meta.png)

为了保存 field 和 value 值，将会再创建一个k-v，格式如下：

![](pika-hash-kv.png)

后创建的k-v 存储了field 和 value。

## 命令操作

为了更好的了解hash 的操作，下面对几类命令逐个学习：

### 创建/更新操作

例如 Hset 操作：

```cpp
Status RedisHashes::HSet(const Slice& key, const Slice& field,
                         const Slice& value, int32_t* res) {
  rocksdb::WriteBatch batch;
  // 此操作需要加锁
  // 函数结束，锁解除
  ScopeRecordLock l(lock_mgr_, key);

  int32_t version = 0;
  uint32_t statistic = 0;
  std::string meta_value;
  // 获取meta 数据
  Status s = db_->Get(default_read_options_, handles_[0], key, &meta_value);
  if (s.ok()) {
    ParsedHashesMetaValue parsed_hashes_meta_value(&meta_value);

    if (parsed_hashes_meta_value.IsStale()
      || parsed_hashes_meta_value.count() == 0) {
      // 如果meta 存在，但是没有用到
      // 则直接更新meta & field & value
      version = parsed_hashes_meta_value.InitialMetaValue();
      parsed_hashes_meta_value.set_count(1);
      batch.Put(handles_[0], key, meta_value);
      HashesDataKey data_key(key, version, field);
      batch.Put(handles_[1], data_key.Encode(), value);
      *res = 1;
    } else {
      // 如果存在，且时间未过期, 版本正确
      version = parsed_hashes_meta_value.version();
      std::string data_value;
      HashesDataKey hashes_data_key(key, version, field);

      // 获取field 数据
      s = db_->Get(default_read_options_,
          handles_[1], hashes_data_key.Encode(), &data_value);
      if (s.ok()) {
        // 如果当前存的field 数据正确
        *res = 0;
        if (data_value == value.ToString()) { // 值也相等，则不操作
          return Status::OK();
        } else {
          // 修改kv
          batch.Put(handles_[1], hashes_data_key.Encode(), value);
          statistic++;
        }
      } else if (s.IsNotFound()) {
          // 如果没有存在kv, 则添加，并更新meta
        parsed_hashes_meta_value.ModifyCount(1);
        batch.Put(handles_[0], key, meta_value);
        batch.Put(handles_[1], hashes_data_key.Encode(), value);
        *res = 1;
      } else {
        // 获取失败
        return s;
      }
    }
  } else if (s.IsNotFound()) {
    // 若meta 未找到, 编码，写入
    char str[4];
    EncodeFixed32(str, 1);
    HashesMetaValue meta_value(std::string(str, sizeof(int32_t)));
    version = meta_value.UpdateVersion();
    batch.Put(handles_[0], key, meta_value.Encode());
    HashesDataKey data_key(key, version, field);
    batch.Put(handles_[1], data_key.Encode(), value);
    *res = 1;
  } else {
    return s;
  }

  // 最后批量写
  s = db_->Write(default_write_options_, &batch);
  // 更新总的统计信息
  UpdateSpecificKeyStatistics(key.ToString(), statistic);
  return s;
}
```

可以看出，在做 HSet 操作时，会对 metadata 和 field value 同时操作，并需要同时更新，而且由于Pika是多线程服务，需要加锁操作。**在频繁访问同一个hash中的数据时，其锁粒度是一个hash的key，可能会有大量的锁冲突出现**。

### 读操作

```cpp
Status RedisHashes::HGet(const Slice& key, const Slice& field,
                         std::string* value) {
  std::string meta_value;
  int32_t version = 0;
  rocksdb::ReadOptions read_options;
  const rocksdb::Snapshot* snapshot;
  ScopeSnapshot ss(db_, &snapshot);
  read_options.snapshot = snapshot;

  // 获取meta 数据
  Status s = db_->Get(read_options, handles_[0], key, &meta_value);
  if (s.ok()) {
    ParsedHashesMetaValue parsed_hashes_meta_value(&meta_value);
    if (parsed_hashes_meta_value.IsStale()) {
    // 如果存在meta，且生效
      return Status::NotFound("Stale");
    } else if (parsed_hashes_meta_value.count() == 0) {
      return Status::NotFound();
    } else {
      // 获取key 值
      version = parsed_hashes_meta_value.version();
      HashesDataKey data_key(key, version, field);
      s = db_->Get(read_options, handles_[1], data_key.Encode(), value);
    }
  }
  return s;
}
```

读操作比较简单，无需加锁。先获取metadata值，再通过metadata 计算出 field 存储key，返回结果即可。

### 删除操作

删除操作有两种，一种是删除整个hash 表(DEL)，一种是删除一个field(HDEL)。首先看下删除整个hash 表的操作。

#### DEL

```cpp
Status RedisHashes::Del(const Slice& key) {
  std::string meta_value;
  ScopeRecordLock l(lock_mgr_, key); // 删除操作需要加锁
  Status s = db_->Get(default_read_options_, handles_[0], key, &meta_value);
  // 获取 metadata 值
  if (s.ok()) {
    ParsedHashesMetaValue parsed_hashes_meta_value(&meta_value);

    if (parsed_hashes_meta_value.IsStale()) {
      // 如果失效了
      return Status::NotFound("Stale");
    } else if (parsed_hashes_meta_value.count() == 0) {
      // 无值
      return Status::NotFound();
    } else {
      // 更新统计值，更新meta_value 即可。 
      uint32_t statistic = parsed_hashes_meta_value.count();
      parsed_hashes_meta_value.InitialMetaValue();
      s = db_->Put(default_write_options_, handles_[0], key, meta_value);
      UpdateSpecificKeyStatistics(key.ToString(), statistic);
    }
  }
  return s;
}
```
这里有个需要注意的地方，hash 删表，并不是删除所有数据，只是把meta_value 值更新即可。(修改count值，时间戳，以及version值)
由于field的key 是通过metadata 中的版本值计算出来的，由于meta_value 版本更新，所有 field value 均失效。
**这个是pika 的一个特性，叫秒删功能。顾名思义，可以做到快速删除hash值，由于其删除hash 只重置了meta值，而hash数据结构已经存在的kv在进行compact时进行**

#### HDEL
下面是删除一个field 的方法:


```cpp
// 从参数可以看出 HDEL 是支持同时删除多个field 的
Status RedisHashes::HDel(const Slice& key,
                         const std::vector<std::string>& fields,
                         int32_t* ret) {
  uint32_t statistic = 0;
  std::vector<std::string> filtered_fields;
  std::unordered_set<std::string> field_set;

  // field 去重
  for (auto iter = fields.begin(); iter != fields.end(); ++iter) {
    std::string field = *iter;
    if (field_set.find(field) == field_set.end()) {
      field_set.insert(field);
      filtered_fields.push_back(*iter);
    }
  }

  rocksdb::WriteBatch batch;
  rocksdb::ReadOptions read_options;
  const rocksdb::Snapshot* snapshot;

  std::string meta_value;
  int32_t del_cnt = 0;
  int32_t version = 0;

  // 加锁
  ScopeRecordLock l(lock_mgr_, key);
  ScopeSnapshot ss(db_, &snapshot);
  read_options.snapshot = snapshot;
  Status s = db_->Get(read_options, handles_[0], key, &meta_value);
  if (s.ok()) {
    ParsedHashesMetaValue parsed_hashes_meta_value(&meta_value);
    if (parsed_hashes_meta_value.IsStale()
      || parsed_hashes_meta_value.count() == 0) {
      *ret = 0;
      return Status::OK();
    } else {
      std::string data_value;
      version = parsed_hashes_meta_value.version();
      // 遍历所有数据，并删除
      for (const auto& field : filtered_fields) {
        HashesDataKey hashes_data_key(key, version, field);
        s = db_->Get(read_options, handles_[1],
                hashes_data_key.Encode(), &data_value);
        if (s.ok()) {
          del_cnt++;
          statistic++;
          batch.Delete(handles_[1], hashes_data_key.Encode());
        } else if (s.IsNotFound()) {
          continue;
        } else {
          return s;
        }
      }
      *ret = del_cnt;
      parsed_hashes_meta_value.ModifyCount(-del_cnt);
      batch.Put(handles_[0], key, meta_value);
    }
  } else if (s.IsNotFound()) {
    *ret = 0;
    return Status::OK();
  } else {
    return s;
  }
  s = db_->Write(default_write_options_, &batch);
  UpdateSpecificKeyStatistics(key.ToString(), statistic);
  return s;
}
```

HDEL 操作支持批量操作。

### 数据的清理

上面提到了，Pika 对于 hash 做了秒删的功能，那秒删之后field中的数据，如何做清理工作呢？
经过研究，发现其实pika没有做主动删除的逻辑，只是通过RocksDB 在做compaction（数据压缩）时调用filter 来实现的。
RocksDB 的compaction, 主要是为了压缩内存和硬盘的使用空间，提升查找速度(LSM 树便是不断的把树结构做merge，做内存落盘和数据压缩）。在copact操作时，提供了可定制的filter 接口。在pika 中，就是通过实现该接口来做秒删功能的。

```cpp
// 针对存储数据的k-v 结构的过滤 （还有一种meta 数据过滤的方法）
class BaseDataFilter : public rocksdb::CompactionFilter {
 public:
  BaseDataFilter(rocksdb::DB* db,
                 std::vector<rocksdb::ColumnFamilyHandle*>* cf_handles_ptr) :
    db_(db),
    cf_handles_ptr_(cf_handles_ptr),
    cur_key_(""),
    meta_not_found_(false),
    cur_meta_version_(0),
    cur_meta_timestamp_(0) {}

  bool Filter(int level, const Slice& key,
              const rocksdb::Slice& value,
              std::string* new_value, bool* value_changed) const override {
    ParsedBaseDataKey parsed_base_data_key(key);
    Trace("==========================START==========================");
    Trace("[DataFilter], key: %s, data = %s, version = %d",
          parsed_base_data_key.key().ToString().c_str(),
          parsed_base_data_key.data().ToString().c_str(),
          parsed_base_data_key.version());

    // 如果是复杂数据结构的key, 两个值不相等，需要取meta中的版本和时间戳
    if (parsed_base_data_key.key().ToString() != cur_key_) {
      cur_key_ = parsed_base_data_key.key().ToString();
      std::string meta_value;
      // destroyed when close the database, Reserve Current key value
      if (cf_handles_ptr_->size() == 0) {
        return false;
      }
      // 基于datakey，算出metakey
      // 查看meta 的状态
      Status s = db_->Get(default_read_options_,
              (*cf_handles_ptr_)[0], cur_key_, &meta_value);
      if (s.ok()) {
        meta_not_found_ = false;
        ParsedBaseMetaValue parsed_base_meta_value(&meta_value);
        cur_meta_version_ = parsed_base_meta_value.version();
        cur_meta_timestamp_ = parsed_base_meta_value.timestamp();
      } else if (s.IsNotFound()) {
        meta_not_found_ = true;
      } else {
        cur_key_ = "";
        Trace("Reserve[Get meta_key faild]");
        return false;
      }
    }

    if (meta_not_found_) {
      Trace("Drop[Meta key not exist]");
      return true;
    }

    //判断版本和过期时间
    int64_t unix_time;
    rocksdb::Env::Default()->GetCurrentTime(&unix_time);
    if (cur_meta_timestamp_ != 0
      && cur_meta_timestamp_ < static_cast<int32_t>(unix_time)) {
      Trace("Drop[Timeout]");
      return true;
    }

    if (cur_meta_version_ > parsed_base_data_key.version()) {
      Trace("Drop[data_key_version < cur_meta_version]");
      return true;
    } else {
      Trace("Reserve[data_key_version == cur_meta_version]");
      return false;
    }
  }

  const char* Name() const override { return "BaseDataFilter"; }

 private:
  rocksdb::DB* db_;
  std::vector<rocksdb::ColumnFamilyHandle*>* cf_handles_ptr_;
  rocksdb::ReadOptions default_read_options_;
  mutable std::string cur_key_;
  mutable bool meta_not_found_;
  mutable int32_t cur_meta_version_;
  mutable int32_t cur_meta_timestamp_;
};
```

从上述代码中可以看出，其实在pika中，对于大批量的数据(比如list，hash，set 等数据结构)均是具有秒删功能的，使用秒删功能比直接删除一方面可以节省执行时间,另一方面可以减少内存碎片，算是以空间换时间的典型例子了。

### 数据扫描

hash 结构除了需要做kv操作外，还有类似扫描key 的操作(`HKEYS`, `HVALS`, `HGETALL`, `HSCAN`)。例如 `HKEYS` 命令会返回所有hash 结构中的 fields. 在 pika 中是如何解决该问题的? 问题的解决需要我们从pika 依赖的RocksDB 中找到答案。

由于 RocksDB 是 基于 LSM 树实现的存储引擎，其 KEY 是有序的，因此，可以通过 RocksDB 的区间查询操作做数据查询。 对于 **HKEYS**, **HVALS**, **HGETALL** 操作，会扫描 HASH 中的所有值，因此其扫描的数据，是 (keySize + key + version) 为前缀做索引前缀; 对于 HSCAN 则在 (keySize + key + version) 的基础上，增加 HSCAN 提供的前缀信息做前缀搜索即可。

## 学习小结

- pika 中，可以存在相同的key 不同存储类型的数据。
- hash 存储值不超过 2^32 , 由于 hash size 存储在 4bytes 的空间中。
- 从Hset中，可以看到，在设计数据结构时，尽量减小 hash 中key 值的数量，减少锁meta的时间。
- pika hash 结构具有秒删功能，对于大批量数据的hash 结果，删除操作和正常命令一样会快速执行。（这个和redis有一定区别）
- pika 中异步删除策略是依赖于RocksDB 的compaction 提供的filter 接口实现的。
- 令人惊喜的是，也有go版本LSM树的实现。([moss](https://github.com/couchbase/moss)) 
- 除了pika外，最近比较火的TiDB的底层存储也是使用的 RocksDB 实现的。

> 备注：本文源码来自 [github.com/Qihoo360/blackwidow](https://github.com/Qihoo360/blackwidow) 中。
