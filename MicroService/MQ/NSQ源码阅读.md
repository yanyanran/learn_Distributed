### NSQ源码阅读

- **执行主逻辑**

在nsq/apps/nsqd/main.go中启动service。

通过第三方svc包进行优雅后台进程管理。svc.Run-> svc.Init-> svc.Start，完成初始化配置（`opts，cfg`）-> 加载历史数据（`nsqd.LoadMetadata`）-> 持久化最新数据（`nsqd.PerisistMetadata`）后开启协程进入主逻辑**nsqd.Main**函数。



Main中除了开启三个处理server和client之间连接（TCP、HTTP、HTTPS）的协程之外，还开启了两[三]个并行的Loop处理协程：**queueScanLoop、lookupLoop、[statsdLoop]**



#### queueScanLoop

> 在单个 Goroutine 中运行，以处理正在进行的和延迟的优先级队列。它管理一个并发处理通道的 queueScanWorker 池（可配置的 QueueScanWorkerPoolMax 最大值（默认：4））。它复制了 Redis 的概率过期算法：它唤醒每个 QueueScanInterval（默认值：100ms）以从本地缓存列表中选择一个随机的 QueueScanSelectionCount（默认值：20）通道（每 QueueScanRefreshInterval 刷新一次（默认值：5s））。如果任何一个队列有工作要做，则通道被视为“dirty”。如果所选通道的 QueueScanDirtyPercent（默认值：25%）dirty，则循环将继续而不休眠。
>
> - > ##### NewTicker
>   >
>   > ###### 返回一个新的 Ticker，其中包含一个通道，该通道将在每次逐笔报价后发送通道上的当前时间。即时报价的周期由持续时间参数指定。股票代码将调整时间间隔或丢弃即时报价以弥补缓慢的接收器。持续时间 d 必须大于零;如果没有，NewTicker将恐慌。停止代码以释放关联的资源。

queueScanLoop主要处理incoming消息的到来和消息的延迟发送。

此协程中维护了一个池，可以通过resizePool来调整池子的大小（大小初始为0）。其中在resizePool中又维护了一个工作协程queueScanWorker来从queueScanLoop中以channel形式接收工作，然后去处理延迟以及正在进行中的队列。

通过从inFightQueue和deferredQueue中读取返回的bool值来定义dirty的bool值，然后把dirty值发送给传参传进来的responChan中，也就是最初在这个协程最上层的queueScanLoop中创建的管道。



延迟发送队列主要使用go中封装的heap包来实现***（后期可优化）***

