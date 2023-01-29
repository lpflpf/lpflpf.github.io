---
title: "服务发现"
date: 2023-01-29T10:51:20+08:00
tags:
  - 服务发现
  - 微服务
  - service discovery
---

## 调研结果
- eureka 已不维护，排除
- nacos，consul 都提供了丰富的功能。
  - nacos 健康检查是provider 心跳；consul 是服务端做心跳。
  - 都提供了dns, http api 的发现功能
  - 都有详细metrix监控能力
  - nacos 有命名空间，consul 开源版本不支持
  - 都可以做鉴权
  - 运维成本：都存在一定运维成本。
- qcm 维护成本低。提供了基本的服务发现功能
- nacos, consul,qcm 都提供了 配置管理功能。
  - qcm，nacos watch 回调方式，qcm 使用的主动请求
- core-dns 无需入侵代码。其他均有代码入侵
  - core-dns 不支持集群外的服务发现，其他服务支持
  - core-dns 对细粒度的心跳检测做的不太好。（比如针对某个接口的POST操作等）
- **个人建议**
  - 在K8S集群内部做服务发现，不建议引入其他服务发现的依赖，使用core-dns即可
  - 如果需要做集群外部的服务依赖，或者有特殊的心跳检查，建议使用nacos。

## 微服务重的服务发现模式

下图是微服务下的各类模式。其中，服务发现仅是各系统间沟通的一种模式。


服务发现一般有几种模式：

- **客户端的服务发现** （通过查询服务注册表发现服务） 比较典型
- **服务端的服务发现** （服务端：一般是中间路由器。查询服务注册表发现服务） 比如 K8s 的  kube-proxy 就是这个路由器的角色
- 客户端或者服务端所需要的注册表，一般是通过 etcd， consul，nacos 等组件完成的。
- 如何注册服务
  - 自注册模式  （自己的服务，启动后主动注册）
  - 第三方注册 （通过第三方服务，做注册和注销，例如通过启动服务时的docker容器做注册）

![MicroservicePatternLanguage.jpg](MicroservicePatternLanguage.jpg)

## 主流注册发现框架技术选型
@2022-3-19 
| | [CoreDNS](github.com/coredns/coredns)|[Eureka](github.com/Netflix/eureka)|[nacos](github.com/alibaba/nacos)|[consul](https://github.com/hashicorp/consul)
|---|---|---|---|---|
|实现语言|Go|Java|Java|go/ client 丰富|Go |
|github Star|8940|11137|21694|24.4k|
|出现时间| |2014|2018|2014|
|一致性原则|- |AP AP|CP|CP,可以弱一致stale|
|实现原理|[coreDNS工作原理](#coreDNS工作原理) |[Eureka工作原理](#Eureka工作原理)|[Nacos工作原理](#Nacos工作原理)|[consul工作原理](#consul工作原理)|
|服务注册发现| kube-proxy watch api-server | http 心跳注册、本地缓存| http 心跳、本地内存缓存、文件缓存| 主动请求、 http,dns 方式的发现|
| 权限管理|	基于k8s 命名空间管理 |	无|	租户隔离，集群隔离|	acl 控制，使用hcl 语言定义|
| 代码入侵 | 无 | api 或者组件 | 通过api注册，或者使用SDK，或者DNS服务发现 | provider: 依赖sdk 注册服务或 dns 获取svc; consumer：无需操作。 通过dns或者配合负载均衡实现|
|负载均衡策略 |rr轮询 wrr 带权重轮询 lc 最小连接法 wlc 带权重最小连接法 目标散列等| 默认RR 客户端可定制|| |
| 实践方式| a) service 使用headless 模式，dns 直接获取对端pod ip 列表 <br/> b) service clusterip 模式，DNS 获取 clusterip,并作NAT后（ipvs/iptables）转发到pod | 容器化部署，支持集群外API注册 <br/> 心跳模式 | 支持心跳模式&主动探测 | 双层结构，分为server 和client。 <br/> client 服务hold 链接，并转发请求 | 
| 客户端|-| Go Client (Star:49)<br/> 支持Rest 方式注册 | go client|Go Client | 
| 集群外负载均衡 | ❌ | ✔ | ✔ |  ✔ |
| 交互性（UI配置） | ❌ | ❌ | ✔ | ✔ | 
| 配置管理 | - | ❌ | ✔  | ✔ | 
| 雪崩保护 | - | ✔ | ✔ | ❌ |
|其他 | k8s 自带<br/> CNCF项目| <ul><li>CNCF项目 </li><li> Netflix 公司开源</li> <li>现在已经不维护了,闭源 2021.10</li> <li>调度可定制化</li></ul>| <ul><li>CNCF项目</li><li>阿里开源</li></ul>| <ul> <li>hashicorp开源项目</li> <li>支持KV存储</li> <li>多数据中心支持</li> <li>没有命名空间概念（需付费）</li> </ul>|


### <font id="coreDNS工作原理">coreDNS 工作原理</font>

ipvs 保存了vip 到 pod 的映射
![](coreDNS.png)
### <font id="Eureka工作原理">Eureka工作原理</font>

1. **Provider**： 发送HTTP请求注册，每30s 做一次续租；90s未续租则下线服务；通过发送cancel 请求，注销服务。
2. **Consumer**: 请求服务获取列表，并缓存。每30s 获取增量数据；当请求不正确时，需要再次请求Provider，获取增量信息 【需要防止雪崩】
3. **最坏情况**：最长可能会有2分钟数据不一致的情况。
4. **自我保护**：如果突然出现大量(15%)不可用的服务，防止大量频繁调用，主动停止所有实例。

### <font id="Nacos工作原理">Nacos工作原理</font>

- **Provider**
![nacos-privider](nacos-privider.png) 
- **consumer**
![nacos-consumer](nacos-consumer.png) 


- 其他
  - 支持临时节点方式的健康检查
  - 支持主动探测方式的健康检查 （服务端ip不经常变，比较试用）
  - 可以与coredns 结合。
、

- 优劣
  - 控制台依赖Mysql
  - 健康检查模式有优势，支持主动探测
  - 针对不同协议有支持
  - 比较好的业务隔离性
### <font id="consul工作原理"></font>

1. consul 和其他服务发现产品大同小异。
2. consul 可以与nginx 结合，动态调整upstream 的策略

## 一致性的探讨   cp OR ap

- **AP 模式**下，遇到分区异常时可能出现数据不一致的情况。该情况下服务时可用的，在服务发现的模式下，比较适用。
- **CP 模式**下，遇到分区异常，会导致服务不可用，知道恢复后才能提供服务。该情况下，可以通过由ip转为dns解析的方式解决。

## 参考：

- [landscape.cncf.io](http://landscape.cncf.io)
- https://microservices.io/
- [ipvs on k8s](https://kubernetes.io/blog/2018/07/09/ipvs-based-in-cluster-load-balancing-deep-dive/)
- [eureka 实现原理](https://github.com/Netflix/eureka/wiki/Understanding-eureka-client-server-communication)
- [nacos 分享  coredns 结合供图](https://github.com/lkxiaolou/lkxiaolou/blob/main/%E6%9C%8D%E5%8A%A1%E5%8F%91%E7%8E%B0/%E6%88%91%E5%9C%A8%E7%BB%84%E5%86%85%E7%9A%84Nacos%E5%88%86%E4%BA%AB/%E6%88%91%E5%9C%A8%E7%BB%84%E5%86%85%E7%9A%84Nacos%E5%88%86%E4%BA%AB.md)
