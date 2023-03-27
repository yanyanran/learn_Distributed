# Distributed Tracing 分布式追踪随笔

最近看社区了解到了分布式追踪，对里面的微服务模型很感兴趣。分布式追踪可以展现出相关服务间的层次关系以及任务的串行或者并行调用关系，我们可以通过追踪一个用户的请求访问链路去观察底层的调用情况，然后就可以发现哪些操作最为耗时然后对其进行优化。

### Trace链路 Span操作 SpanContext信息

Trace是由多个Span组成的一个有向无环图，而Span可以理解为一个单个操作，span之间通过嵌套或者顺序排列建立因果关系。

> Span 包含：
>
> - 操作名称 Name ：这个名称简单，可读性高
> - 起始时间和结束时间：操作的生命周期
> - 属性 Attributes:：一组<K,V>键值对构成的集合。值可以是字符串、布尔或者数字类型，一些链路追踪系统也称为 Tags
> - 事件 Event
> - 上下文 SpanContext:：Span的上下文信息
> - 链接 Links：描述 Span 节点关系的连线，它的描述信息保存在 SpanContext 中

其中SpanContext就是在一个Trace中把目前的Span和Trace的相关信息传递到下一级Span中去。

> SpanContext包括：
>
> - TraceId：随机 16 字节数组（4bf92f3577b34da6a3ce929d0e0e4736
> - SpanId：随机 8 字节数组（00f067aa0ba902b7
> - Baggage Items ：存在于 Trace的 一个键值对集合，也需要 Span 间传输

以最近研究的打车软件Uber所使用的后台Jaeger为例，当我在客户端大量频繁点击发起多次打车，后台u追踪UI如下：

![](https://github.com/yanyanran/pictures/blob/main/Tracing.png?raw=true)

上图中的左侧列就是trace和span的结构关系，中间的时间条表示一个span的生命周期，下面的一系列‘Tags/Process/Logs‘表示的是Span的相关属性。

从上图可以观察出操作的瓶颈在于对mysql的查询，相比于下面的redis查询时间简直是翻倍又翻倍。



### Trace链路传递

在一个链路追踪过程中会有多个Span操作，这时候我们希望把调用链状态在Span中传递下去保存下来，这时候SpanContext就会封装一个KV键值对集合，然后将数据像打包行李一样打包下去。

在OpenTelemetry中称为**Baggage**，它会在一条追踪链路中的所有Span内**全局传输**，于是可以通过这个Baggage实现强大的追踪功能。

> *（但因为它是全局，所以它会拷贝到每一个本地以及远程的子Span，如果数量太大的话会降低系统的吞吐量或者是RPC的延迟）*

出了这种全局方式的传递监控，我们还可以用打Tags的方式，其实Tags的本质就还是Span的属性（KV）。Span 的 Tags 可以用来记录业务相关的数据，并存储于追踪系统中（但它不继承也不传输）