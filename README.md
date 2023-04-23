# Distributed_learning
##### 学习分布式的记录历程

| project                                                      | Introduction                                                 |
| ------------------------------------------------------------ | :----------------------------------------------------------- |
| [MyRPC](https://github.com/yanyanran/learn_Distributed/tree/main/MyRPC) | 用Go实现的一个rpc框架。此框架包括rpc服务端、支持并发的客户端以及一个简易的服务注册和发现中心；支持选择不同的序列化与反序列化方式；为防止服务挂死，添加了超时处理机制；支持 TCP、Unix、HTTP 等多种传输协议；支持多种负载均衡模式。 |
| [MyCache](https://github.com/yanyanran/learn_Distributed/tree/main/MyCache) | 用Go实现的一个分布式缓存。实现了 LRU 缓存淘汰算法去解决资源限制的问题；实现了单机并发，并给用户提供了自定义数据源的回调函数；实现了一致性哈希算法去解决远程节点的挑选问题；同时创建了HTTP 客户端和HTTP服务端，实现多节点间的通信；解决缓存击穿的问题；使用 protobuf 库优化了节点间通信性能。 |

------

开始接触云原生微服务相关，研究Otel的分布式追踪

研究微服务中间件，消息队列，接触Dubbo Sercurity鉴权部分
