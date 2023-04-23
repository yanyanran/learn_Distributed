# NSQ源码阅读

- ## **执行主逻辑**

在nsq/apps/nsqd/main.go中启动service。

通过第三方svc包进行优雅后台进程管理。在main中定义了一个自定义struct--program，然后通过让program实例去实现svc.Service中的接口函数。

svc.Run（启动守护进程）-> svc.Init-> svc.Start，**完成初始化配置**（`opts，cfg`）-> **加载历史元数据**（`nsqd.LoadMetadata`）-> **持久化最新数据到磁盘**（`nsqd.PerisistMetadata`）后开启协程进入主逻辑**nsqd.Main**函数，启动nsqd实例。此时svc.Run内部进入阻塞状态，启动守护程序：

```go
for {
   select {
   case s := <-signalChan:
      if h, ok := service.(Handler); ok {
         if err := h.Handle(s); err == ErrStop {
            goto stop
         }
      } else {
         goto stop
      }
       // service不关闭
   case <-ctx.Done():
      goto stop
   }
}
```

Main中除了开启三个处理server和client之间连接（TCP、HTTP、HTTPS）的协程之外，还开启了两[三]个并行的Loop处理协程：**queueScanLoop、lookupLoop、[statsdLoop]**



在退出nsqd服务时，nsqd会将当时的topic和channel信息持久化到磁盘【包含所有topic和各个topic的包含的channel以及是否被暂停的信息】以便在下次启动时可以重新加载这些信息【以json文本的形式存储在磁盘中】

所以在启动nsqd时会先加载一次元数据（nsqd.dat），





------

## queueScanLoop  循环监控队列信息

queueScanLoop主要**处理infight和deferred的优先级队列**。

每个channel中都维护了inFight【已发未收】和deferred【延迟发送】两个队列



> - ##### NewTicker
>
>   ###### 返回一个新的 Ticker，其中包含一个通道，该通道将在每次逐笔报价后发送通道上的当前时间。即时报价的周期由持续时间参数指定。股票代码将调整时间间隔或丢弃即时报价以弥补缓慢的接收器。持续时间 d 必须大于零;如果没有，NewTicker将恐慌。停止代码以释放关联的资源。



#### 动态调整worker数量

而且这个pool内的worker数量是随着**channel数量**动态调整的，每隔n（默认5s）秒就会去更新一次worker数量。

queueScanLoop中维护的三个chan就是为了动态调整worker数量的：

```go
func (n *NSQD) queueScanLoop() {
   workCh := make(chan *Channel, n.getOpts().QueueScanSelectionCount)  // 存放随机选择的channel
   responseCh := make(chan bool, n.getOpts().QueueScanSelectionCount) // 收到dirty-> true/false
   closeCh := make(chan int)
    
    // ...
}
```

所有worker都会监听closeCh，当worker多时，queueScanLoop往closeCh中发送一个退出消息，任何一个worker在监听到退出消息时结束退出；当worker数量过少时，则启动一个新的worker。



####  ***Redis 的概率过期算法***

每隔QueueScanInterval秒（默认100s)从nsqd包含的所有channel中随机选择QueueScanSelectionCount个（默认20个）进行处理。如果在选择的20个channel中超过QueueScanDirtyPercent（默认25%）的channel需要处理inFlight或者Deferred队列，则在处理完这些channel之后，继续选择20个channel处理它们的inFlight和Deferred队列。直到不足25%，queueScanLoop会停下来，等待下一个100s时间的到来。



#### Loop处理过程

***queueScanLoop***内部维护了一个*协程池queueScanWorkerPool*，，可以通过***resizePool***来调整池内的worker的数量（初始为0），在queueScanLoop开始for循环之前会有一次初始化，然后每隔QueueScanRefreshInterval秒（初始为5s）重新调resizePool调整一次worker数。

其中在resizePool中又维护了**工作协程queueScanWorker**来从queueScanLoop中以channel形式**接收工作，然后去处理延迟以及正在进行中的队列**。每个queueScanWorker都是一个协程，这些worker的工作就是上面说的过期算法中处理相关队列的。



Loop将随机选择的channel放入workCh中，任何一个worker在监听到workCh有消息时拿到一个channel进行处理，处理的结果（dirty）通过**responseCh**返回给queueScanLoop，以便queueScanLoop决策是否要继续循环处理剩余的channel。





## PQueue优先级队列设计



> deferred队列使用的是go中封装的heap包来实现的优先队列
>
> 而infight队列是自己实现的优先队列（但其实原理和heap包一样）



使用最小**堆**的数据结构，优先队列采用类似于树结构，节点index递增方法为层序顺序，**index越小【距离超时时间越近的】优先级越高**，pop的对象为树的根节点。

暴露在外最常用的方法是PeekAndShift，它主要是**查看已发送【inFight】队列 并获取队头消息**。首先判断消息是否超时，超时直接pop，没超时就return还有多久超时。在pop后会调用swap、up、down这些方法去维持堆的正确结构。

这种最小堆的模型有一个好处就是能够根据堆顶来判断整个队列的超时情况。假如堆头都没有超时的话，那么后面的也必不会超时。





## nsqd设计

nsqd被引用于：

​	tcp、http、protocol、client、channel、topic、main/program

![](https://github.com/yanyanran/pictures/blob/main/nsqd.png?raw=true)

nsqd启动





------

channel包含多个client的信息【一个channel可以有多consumer】

一个consumer就是一个TCP连接，所以在*Topic*中有一个变量**channelMap**，它是一个字典，存储了**channel名称和Channel实例**的映射关系；

在*Channel*中有一个变量**clients**，这也是一个字典，存储了**clientID和consumer[clientV2实例]**之间的映射。

```consumer在实际是一个接口，由于clientV2实现了这个接口，所以channel实际上存储的是clientID和clientV2实例的映射```



## Topic设计





## Channel设计







------



## nsqlookupd

![](https://github.com/yanyanran/pictures/blob/main/nsqdlookup.png?raw=true)

