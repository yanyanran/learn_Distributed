# NSQ源码阅读



## nsqd设计

nsqd被引用于：

​	tcp、http、protocol、client、channel、topic、main/program

![](https://github.com/yanyanran/pictures/blob/main/nsqd.png?raw=true)

#### nsqd启动

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

所以在启动nsqd时会先加载一次元数据（nsqd.dat），并且会生成一份新的元数据覆盖原始元数据。



#### nsqd接收消息

###### 1、HTTP方式接收

主要通过在newHTTPServer中配置路由和handler，http中接收消息的handler是doPUB：

```go
unc (s *httpServer) doPUB(w http.ResponseWriter, req *http.Request, ps httprouter.Params) (interface{}, error) {
	// .....
    
   // 从request中获取msg body
   body, err := io.ReadAll(io.LimitReader(req.Body, readMax))
    
   // .....

   // 从request中获取【topic name】，然后根据topic name从nsqd实例中获取topic实例（没有此topic就新建一个
   reqParams, topic, err := s.getTopicFromQuery(req)
   if err != nil {
      return nil, err
   }

   var deferred time.Duration
   // 请求中可以带上defer参数，表示延迟时间
   if ds, ok := reqParams["defer"]; ok {
      var di int64
      di, err = strconv.ParseInt(ds[0], 10, 64)
      if err != nil {
         return nil, http_api.Err{400, "INVALID_DEFER"}
      }
      deferred = time.Duration(di) * time.Millisecond
      if deferred < 0 || deferred > s.nsqd.getOpts().MaxReqTimeout {
         return nil, http_api.Err{400, "INVALID_DEFER"}
      }
   }

   // 生成一个唯一id，构建Message对象
   msg := NewMessage(topic.GenerateID(), body)
   msg.deferred = deferred
   // 将消息加入topic中
   err = topic.PutMessage(msg)
   if err != nil {
      return nil, http_api.Err{503, "EXITING"}
   }

   return "OK", nil
}
```

从客户端的request中解析出topic name和msg body，然后获取Topic实例（Topic不存在就新建一个），再根据msg body和新建的唯一uuid创建Message实例（有延迟发送时间就注册进去），最后将消息存入Topic中。



#### 线程安全的GetTopic设计

使用读写锁保证了线程安全：RLock加**读锁**时，**不限制读但限制写**

GetTopic时，先从nsqd的topicMap中获取（此时读锁启动），如果成功获取到就直接return；

没有获取到代表当前topic不存在-> 新建topic（此时写锁启动）。**【此时会再访问一次topicMap】**

```go
func (n *NSQD) GetTopic(topicName string) *Topic { // 线程安全的
   n.RLock()
   t, ok := n.topicMap[topicName]
   n.RUnlock()
   if ok {
      return t
   }

   // 不存在此topic实例，加个写锁
   n.Lock()

   // 再次从topicMap中获取一次。为什么？
   // 考虑一种情况：在某一时刻，同时有线程A B C获取同一个topic D
   // 此时A B C通过n.topicMap[D]没拿到实例，全部走到n.Lock()，
   // A先拿到锁成功创建了D的实例并且添加入topicMap中，然后释放锁返回。
   // B, C中某一个获取到A释放的锁进入临界区，
   // 如果没有再从topicMap中获取一次，则会重新创建一个topic实例，
   // 可能会造成数据丢失
   t, ok = n.topicMap[topicName]
   if ok {
      // 【1】如果此时获取到topic实例，说明几乎在同一时刻有另外一个线程也在获取该topic
      n.Unlock()
      return t
   }
    
   deleteCallback := func(t *Topic) {
      n.DeleteExistingTopic(t.name)
   }
   // 【2】如果第二次没有从topicMap中获取到，则新建topic实例，并添加到topicMap中
   t = NewTopic(topicName, n, deleteCallback)
   n.topicMap[topicName] = t

   // 到这里已经完成了topic实例的新建工作，但是msgPump还没有启动
   n.Unlock()

   n.logf(LOG_INFO, "TOPIC(%s): created", t.name)
   if atomic.LoadInt32(&n.isLoading) == 1 { // atomic.Value 进行结构体字段的并发存取值，保证原子性
      return t
   }

    // 从远端获取channel
   lookupdHTTPAddrs := n.lookupdHTTPAddrs()
   if len(lookupdHTTPAddrs) > 0 {
      channelNames, err := n.ci.GetLookupdTopicChannels(t.name, lookupdHTTPAddrs)
      if err != nil {
         n.logf(LOG_WARN, "failed to query nsqlookupd for channels to pre-create for topic %s - %s", t.name, err)
      }
      for _, channelName := range channelNames {
         if strings.HasSuffix(channelName, "#ephemeral") {
            continue
         }
         t.GetChannel(channelName)
      }
   } else if len(n.getOpts().NSQLookupdTCPAddresses) > 0 {
      n.logf(LOG_ERROR, "no available nsqlookupd to query for channels to pre-create for topic %s", t.name)
   }

   // 开启 topic msgPump
   t.Start()
   return t
}
```



###### 2、TCP方式接收

在TCPServer中启动tcp Handle，通过客户端发送的msg body在Handle中指定协议版本，接着生成连接对象建立连接，在protocol的IOLoop中通信。

```go
func (p *protocolV2) IOLoop(c protocol.Client) error {  // 对tcp连接进行处理
   var err error
   var line []byte
   var zeroTime time.Time

   client := c.(*clientV2)

   messagePumpStartedChan := make(chan bool)
   // 为每个连接启动一个goroutine，通过chan进行消息通信
   // 完成消息接收，消息投递，订阅channel，发送心跳包等工作
   go p.messagePump(client, messagePumpStartedChan) // pump
   // messagePumpStartedChan作为messagePump的参数
   // 用来阻塞当前进程，直到messagePump完成初始化工作，
   // 关闭messagePumpStartedChan后，当前进程才能继续
   <-messagePumpStartedChan

   for {
      if client.HeartbeatInterval > 0 {
         client.SetReadDeadline(time.Now().Add(client.HeartbeatInterval * 2))
      } else {
         client.SetReadDeadline(zeroTime)
      }

      // 读取直到第一次遇到'\n'，
      // 返回缓冲里的包含已读取的数据和'\n'字节的切片
      line, err = client.Reader.ReadSlice('\n')
      if err != nil {
         if err == io.EOF {
            err = nil
         } else {
            err = fmt.Errorf("failed to read command - %s", err)
         }
         break
      }

      // trim the '\n'
      line = line[:len(line)-1]
      // optionally trim the '\r'
      if len(line) > 0 && line[len(line)-1] == '\r' {
         line = line[:len(line)-1]
      }
      // 从数据中解析出命令
      params := bytes.Split(line, separatorBytes)

      p.nsqd.logf(LOG_DEBUG, "PROTOCOL(V2): [%s] %s", client, params)

      var response []byte
      // 执行命令 Exec分配指令对应方法
      response, err = p.Exec(client, params)
      if err != nil {
          // .....
         continue
      }
       
       // .....
   }

    // 出现err
   p.nsqd.logf(LOG_INFO, "PROTOCOL(V2): [%s] exiting ioloop", client)
   close(client.ExitChan)
   if client.Channel != nil {
      client.Channel.RemoveClient(client.ID)
   }

   return err
}
```

Exec就是tcp的命令执行方法，通过switch case来解析命令（PUB、FIN、RDY....）调用对应的消息处理方法。



HTTP和TCP接收消息的本质都差不多，都是先**获取/创建一个topic**，然后**NewMsg创建一个消息对象**，最后**topic.PutMsg将消息放入topic中**。



#### nsqd发送消息

nsqd往客户端发消息主要走的是**tcp连接**，启动了一个tcp listener并连接后，通过tcp Handle调IOLoop去轮询读消息处理消息

而IOLoop内又存在两个工作协程：**Exec**和**MsgPump**，它俩之间通过多个事件Chan去同步信息。

其中Exec主要是通过解析tcp命令获取操作参数（PUB、RDY这些）然后执行相对应的方法*[写管道触发msgPump]*，MsgPump则无限for去监听多个事件管道进行处理。



 其中对bakendMsgChan和memoryMsgChan两个管道的读写操作就是nsqd往客户端发送消息的行为：

```go
func (p *protocolV2) messagePump(client *clientV2, startedChan chan bool) {
    
    // .....(pump的初始化

	for {
		if subChannel == nil || !client.IsReadyForMessages() {
			// 客户端还没准备好接收数据
			memoryMsgChan = nil
			backendMsgChan = nil
			flusherChan = nil
			// force flush
			// 强制刷新一次，将client中Inflight的消息发出去
			client.writeLock.Lock()
			err = client.Flush()
			client.writeLock.Unlock()
			if err != nil {
				goto exit
			}
			flushed = true
		} else if flushed {
			// 将memoryMsgChan, backendMsgChan设置为我们订阅的channel所属的memoryMsgChan和backend
			memoryMsgChan = subChannel.memoryMsgChan
			backendMsgChan = subChannel.backend.ReadChan()
			flusherChan = nil
		} else {
			// buffer中有数据了，设置flusherChan， 保证在OutputBufferTimeout之后可以将消息发送出去
			memoryMsgChan = subChannel.memoryMsgChan
			backendMsgChan = subChannel.backend.ReadChan()
			flusherChan = outputBufferTicker.C
		}

		select {
		// OutputBufferTimeout时间到，刷新缓冲区，将消息发送出去
		case <-flusherChan:
			client.writeLock.Lock()
			err = client.Flush()
			client.writeLock.Unlock()
			if err != nil {
				goto exit
			}
			flushed = true
		case <-client.ReadyStateChan:
		case subChannel = <-subEventChan:
			// you can't SUB anymore
			// 订阅channel(EXEC(SUB))和更新RDY值时（EXEC(RDY)）时
			// 会往subEventChan和client.ReadyStateChan中发送消息。
			// messagePump如果接收从这两个golang channel中接收到消息，
			// 则不可以再订阅nsq channel
			subEventChan = nil
		case identifyData := <-identifyEventChan:
			// 同样的，首次连接时执行EXEC(IDENTIRY)
			// 会将客户端的配置通知到identifyEventChan。
			// messagePump接收到后，根据客户端的配置调整心跳频率等。
			// 之后再发送IDENTIFY，messagePump接收不到
			identifyEventChan = nil

			// 如下，根据客户端配置，调整不同的参数
			outputBufferTicker.Stop()
			if identifyData.OutputBufferTimeout > 0 {
				outputBufferTicker = time.NewTicker(identifyData.OutputBufferTimeout)
			}

			heartbeatTicker.Stop()
			heartbeatChan = nil
			if identifyData.HeartbeatInterval > 0 {
				heartbeatTicker = time.NewTicker(identifyData.HeartbeatInterval)
				heartbeatChan = heartbeatTicker.C
			}

			if identifyData.SampleRate > 0 {
				sampleRate = identifyData.SampleRate
			}

			msgTimeout = identifyData.MsgTimeout
		case <-heartbeatChan:
			// 每隔HeartbeatInterval发送一次心跳包
			err = p.Send(client, frameTypeResponse, heartbeatBytes)
			if err != nil {
				goto exit
			}
		case b := <-backendMsgChan:
			// 如果backend中有数据，channel会将backend的消息从文件中读出，
			// 发送到golang channel，推送至客户端
			if sampleRate > 0 && rand.Int31n(100) > sampleRate {
				continue
			}

			msg, err := decodeMessage(b)
			if err != nil {
				p.nsqd.logf(LOG_ERROR, "failed to decode message - %s", err)
				continue
			}
			// 每投递一次消息，则记录一次，超过一定次数就丢弃
			msg.Attempts++

			// 将消息移到InFlight队列，同时记录该消息在InFlight中的超时时间
			subChannel.StartInFlightTimeout(msg, client.ID, msgTimeout)
			// 增加客户端的InFlightCount和MessageCount
			client.SendingMessage()
			// 将消息写入buffer中
			err = p.SendMessage(client, msg)
			if err != nil {
				goto exit
			}
			// 由于buffer中有数据了，将flushed设置为false，可以打开flusherChan
			// 以便触发client.Flush()将消息发出去
			flushed = false
		case msg := <-memoryMsgChan:
			// 获取memoryMsgChan中的消息与获取backendMsgChan中的基本一致
			if sampleRate > 0 && rand.Int31n(100) > sampleRate {
				continue
			}
			msg.Attempts++

			subChannel.StartInFlightTimeout(msg, client.ID, msgTimeout)
			client.SendingMessage()
			err = p.SendMessage(client, msg)
			if err != nil {
				goto exit
			}
			flushed = false
		case <-client.ExitChan:
			// 接收到exit消息 则退出for
			goto exit
		}
	}

exit:
	p.nsqd.logf(LOG_INFO, "PROTOCOL(V2): [%s] exiting messagePump", client)
	heartbeatTicker.Stop()
	outputBufferTicker.Stop()
	if err != nil {
		p.nsqd.logf(LOG_ERROR, "PROTOCOL(V2): [%s] messagePump error - %s", client, err)
	}
}
```

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









## Topic设计

每个Topic中都维护一个`memoryMsgChan内存队列`（默认长度10000）和`backend磁盘队列`【内存队列满了就存到backend磁盘队列中去】

##### 假如nsqd在运行过程中关闭了某个Topic：

退出的过程中topic会把memoryMsgChan的数据存到backend中，体现了**持久化数据**这个点。



## Channel设计

channel包含多个client的信息【一个channel可以有多consumer】

一个consumer就是一个TCP连接，所以在*Topic*中有一个变量**channelMap**，它是一个字典，存储了**channel名称和Channel实例**的映射关系；

在*Channel*中有一个变量**clients**，这也是一个字典，存储了**clientID和consumer[clientV2实例]**之间的映射。

```consumer在实际是一个接口，由于clientV2实现了这个接口，所以channel实际上存储的是clientID和clientV2实例的映射```







------



## nsqlookupd

![](https://github.com/yanyanran/pictures/blob/main/nsqdlookup.png?raw=true)

