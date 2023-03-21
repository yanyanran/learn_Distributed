# dashboard
url: http://47.96.183.43:3000/d/ewNiYNcVk/dubbo-application-dashboard?orgId=1&refresh=5s
user:  test/test

全局面板：todo 
应用级面板：  http://47.96.183.43:3000/d/ewNiYNcVk/dubbo-application-dashboard?orgId=1&refresh=5s
接口级面板：todo


官方参考文档（修改中）：
https://github.com/songxiaosheng/dubbo-website/blob/master/content/zh-cn/overview/core-features/observability.md

# metrics 命名
https://github.com/apache/dubbo/issues/11321

dubbo_类型_行为_单位_聚合函数


# 采集指标参考

## 应该采集哪些指标
参考四个黄金信号、RED方法、USE方法等理论
- 延迟（Latency）、
- 流量（Traffic）、 
- 错误（Errors） 
- 饱和度（Saturation）

## 采集流程
### 主流程
采集指标 -> 缓存指标样本 -> 部分指标聚合分析 ->指标转换 -> 通过端口输出指标数据
### 采集器采集
- Dubbo应用信息（如应用名、版本号）
- 提供者
- 消费者
- 异常情况
- 注册中心
- 元数据中心
- 配置中心
- 线程池
- 其他
## 指标导出
通过复用QOS端口22222 通过http形式导出

curl http://localhost:22222/metrics

参考案例如下：
https://github.com/apache/dubbo-samples/tree/master/4-governance/dubbo-samples-metrics-prometheus

## 服务发现
指标端口服务 要让普罗米修发现默认配置的静态服务发现 这样不太好要进行动态的服务发现 需要将指标端口 和path注册到注册中心


# 指标上报

- 主动推送的push方式
- 被动拉取方式（非实时性要求推荐这种）
# 指标含义参考
内容可能不全后续补充

| Metrics Name                   | Description              |
| ------------------------------ | ------------------------ |
| dubbo_thread_pool_max_size     | Thread Pool Max Size     |
| dubbo_thread_pool_largest_size | Thread Pool Largest Size |
| dubbo_thread_pool_thread_count | Thread Pool Thread Count |
| dubbo_thread_pool_queue_size   | Thread Pool Queue Size   |
| dubbo_thread_pool_active_size  | Thread Pool Active Size  |
| dubbo_thread_pool_core_size    | Thread Pool Core Size    |
|                                |                          |



| Metrics Name                                     | Description                        | 预期 | 结果 |
| ------------------------------------------------ | ---------------------------------- | ---- | ---- |
| dubbo_consumer_rt_milliseconds_sum               | Sum Response Time                  |      |      |
| dubbo_consumer_rt_milliseconds_p99               | Response Time P99                  |      |      |
| dubbo_consumer_rt_milliseconds_avg               | Average Response Time              |      |      |
| dubbo_consumer_rt_milliseconds_last              | Last Response Time                 |      |      |
| dubbo_consumer_requests_processing               | Processing Requests                |      |      |
| dubbo_consumer_rt_milliseconds_max               | Max Response Time                  |      |      |
| dubbo_consumer_rt_milliseconds_p95               | Response Time P95                  |      |      |
| dubbo_consumer_requests_succeed_aggregate        | Aggregated Succeed Requests        |      |      |
| dubbo_consumer_requests_succeed_total            | Succeed Requests                   |      |      |
| dubbo_consumer_requests_total                    | Total Requests                     |      |      |
| dubbo_consumer_rt_milliseconds_min               | Min Response Time                  |      |      |
| dubbo_consumer_requests_total_aggregate          | Aggregated Total Requests          |      |      |
| dubbo_consumer_qps_total                         | Query Per Seconds                  |      |      |
| dubbo_consumer_requests_failed_total             | Total Failed Requests              |      |      |
| dubbo_consumer_requests_timeout_total            | Total Timeout Failed Requests      |      |      |
| dubbo_consumer_requests_failed_total_aggregate   | Aggregated failed total Requests   |      |      |
| dubbo_consumer_requests_timeout_failed_aggregate | Aggregated timeout Failed Requests |      |      |



| Metrics Name                              | Description                 |
| ----------------------------------------- | --------------------------- |
| dubbo_provider_requests_total             | Total Requests              |
| dubbo_provider_rt_milliseconds_avg        | Average Response Time       |
| dubbo_provider_rt_milliseconds_sum        | Sum Response Time           |
| dubbo_provider_requests_total_aggregate   | Aggregated Total Requests   |
| dubbo_provider_requests_succeed_total     | Succeed Requests            |
| dubbo_provider_qps_total                  | Query Per Seconds           |
| dubbo_provider_requests_processing        | Processing Requests         |
| dubbo_provider_rt_milliseconds_p99        | Response Time P99           |
| dubbo_provider_requests_succeed_aggregate | Aggregated Succeed Requests |
| dubbo_provider_rt_milliseconds_p95        | Response Time P95           |
| dubbo_provider_rt_milliseconds_max        | Max Response Time           |
| dubbo_provider_rt_milliseconds_min        | Min Response Time           |
| dubbo_provider_rt_milliseconds_last       | Last Response Time          |

 



| Metrics Name                                   | Description               | 说明                               | 验证结果 |
| ---------------------------------------------- | ------------------------- | ---------------------------------- | -------- |
| dubbo_register_rt_milliseconds_max             | Max Response Time         | 应用级实例注册总的最大时间         |          |
| dubbo_register_rt_milliseconds_avg             | Average Response Time     | 应用级实例注册总的平均时间         |          |
| dubbo_register_rt_milliseconds_sum             | Sum Response Time         | 应用级实例注册总的注册时间         |          |
| dubbo_register_rt_milliseconds_min             | Min Response Time         | 应用级实例注册总的最小时间         |          |
| dubbo_registry_register_requests_succeed_total | Succeed Register Requests | 应用级实例注册成功的次数           | ok       |
| dubbo_registry_register_requests_total         | Total Register Requests   | 应用级实例注册总次数包含成功与失败 | ok       |
| dubbo_registry_register_requests_failed_total  | Failed Register Requests  | 应用级实例注册失败次数             |          |
| dubbo_register_rt_milliseconds_last            | Last Response Time        | 应用级实例注册最新响应时间         |          |
| dubbo_registry_register_requests_failed_total  | Failed Register Requests  | 应用级实例注册失败次数             |          |

 









元数据指标生效范围：当元数据为集中式配置时（report-metadata为true或者metadataType为remote）



| Metrics Name                               | Description                    | 说明                                                         |
| ------------------------------------------ | ------------------------------ | ------------------------------------------------------------ |
| dubbo_metadata_push_num_total              | Total Num                      | **提供者** 推送元数据到元数据中心的成功次数,当提供者元数据发生了变更时触发 |
| dubbo_metadata_push_num_succeed_total      | Succeed Push Num               | **提供者** 推送元数据到元数据中心的成功次数,当提供者元数据发生了变更时触发 |
| dubbo_metadata_push_num_failed_total       | Failed Push Num                | **提供者** 推送元数据到元数据中心的失败次数,当提供者元数据发生了变更时并且出现异常触发 |
| dubbo_metadata_subscribe_num_total         | Total Metadata Subscribe Num   | **消费者** 获取元数据的总次数，当消费者启动时本地磁盘缓存无元数据获取元数据的次数 |
| dubbo_metadata_subscribe_num_succeed_total | Succeed Metadata Subscribe Num | **消费者** 获取元数据的总次数，当消费者启动时本地磁盘缓存无元数据并且成功获取元数据的次数 |
| dubbo_metadata_subscribe_num_failed_total  | Failed Metadata Subscribe Num  | **消费者** 获取元数据的总次数，当消费者启动时本地磁盘缓存无元数据并且获取元数据失败的次数 |
| dubbo_push_rt_milliseconds_sum             | Sum Response Time              | **提供者：**推送元数据到元数据中心的总时间                   |
| dubbo_push_rt_milliseconds_last            | Last Response Time             | **提供者：**推送元数据到元数据中心的最新耗时                 |
| dubbo_push_rt_milliseconds_min             | Min Response Time              | **提供者：**推送元数据到元数据中心的最小时间                 |
| dubbo_push_rt_milliseconds_max             | Max Response Time              | **提供者：**推送元数据到元数据中心的最大时间                 |
| dubbo_push_rt_milliseconds_avg             | Average Response Time          | **提供者：**推送元数据到元数据中心的平均时间                 |
| dubbo_subscribe_rt_milliseconds_sum        | Sum Response Time              | **消费者：** 获取元数据从元数据中心的总时间                  |
| dubbo_subscribe_rt_milliseconds_last       | Last Response Time             | **消费者：**推送元数据到元数据中心的最新耗时                 |
| dubbo_subscribe_rt_milliseconds_min        | Min Response Time              | **消费者：**推送元数据到元数据中心的最小时间                 |
| dubbo_subscribe_rt_milliseconds_max        | Max Response Time              | **消费者：**推送元数据到元数据中心的最大时间                 |
| dubbo_subscribe_rt_milliseconds_avg        | Average Response Time          | **消费者：**推送元数据到元数据中心的平均时间                 |

 





| Metrics Name                 | Description            |
| ---------------------------- | ---------------------- |
| dubbo_application_info_total | Total Application Info |
|                              |                        |
|                              |                        |
|                              |                        |
|                              |                        |
|                              |                        |
|                              |                        |

 

 
