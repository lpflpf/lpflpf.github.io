---
title: "API 管理工具"
date: 2023-01-28T11:11:44+08:00
tags: 
  - 开发工具
---


>
> 期望的工作流
>
> - 接口定义，提供mock数据
> - 接口开发，提供test工具，测试接口
> - 接口更新，自动更新接口文档
> - 接口测试，提供自动测试工具



## 选型

|工具| |文档管理|私有化|导入|权限|mock|测试 | 语言 |
|--|---|---|---|---|---|---|---|---|
|[swagger](https://github.com/swagger-api/swagger-ui) ||✔|✔|✔|❌|❌|❌|多语言|
|[yapi](https://github.com/YMFE/yapi) | 24.4K|✔|✔|✔|✔|✔|✔|多语言|
|[apidoc/apigen](https://github.com/apidoc/apidoc) | |✔|✔|✔|❌|✔|✔|多语言|
|[easydoc](https://github.com/wuyumin/easydoc)|| ✔|✔|❌|✔|❌|❌|❌| 
|confluence|| ✔|✔|❌|✔|❌|❌|❌|
|[gitbook](https://github.com/GitbookIO/gitbook)| |✔|✔|❌|❌|❌|❌|❌|
|[apifox apipost eolinker](https://github.com/apifox/apifox)| | ✔|❌|✔|✔|✔|✔|多语言|
|[rap2](https://github.com/thx/rap2-delos)|7.3K|✔|✔|❌|✔|✔|❌|多语言|

最终选择yapi做api接口管理工具。

1. **apidoc/apigen**: node 服务，注释→ 文档 ,  go 语言实现
2. **yapi**: 支持swagger导入
3. **easydoc**: swagger 支持生成md文件，easydoc 需要手动修改文档
4. **confluence**: 支持生成md文件，需要手动修改文档
5. **gitbook**: 支持生成md文件，需要手动修改文档
6. **rap2**: 阿里妈妈开发，支持历史记录查看,可以通过结构化数据转为swagger (需要自研）
