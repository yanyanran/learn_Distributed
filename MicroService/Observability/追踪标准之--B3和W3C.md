# 追踪标准之--B3和W3C

1. > *最近接的Dubbo-go项目，负责tracing模块，第一阶段是让tracing支持B3和W3C追踪标准*。

- ## B3

B3最初由Zipkin项目开发，目前成为OpenTracing标准之一。B3标准使用HTTP头传递跟踪信息，以确保分布式系统中所有服务都可以访问该信息。B3使用以下三个HTTP头：X-B3-TraceId、X-B3-SpanId、X-B3-ParentSpanId（以X-B3作为开头，用来跟踪跨服务边界的TraceContext（上下文传播）

- #### 整体流程

最常见的用例：

将TraceContext从发送RPC request的客户端复制到接收它的服务器。

在这种情况下，使用了相同的span ID意味着操作的客户端和服务器端都在跟踪树中的相同节点中结束。

下面是一个使用**多标头编码**的示例流程，假设 HTTP 请求携带传播的跟踪：

```
   Client Tracer                                                  Server Tracer     
┌───────────────────────┐                                       ┌───────────────────────┐
│                       │                                       │                       │
│   TraceContext        │          Http Request Headers         │   TraceContext        │
│ ┌───────────────────┐ │         ┌───────────────────┐         │ ┌───────────────────┐ │
│ │ TraceId           │ │         │ X-B3-TraceId      │         │ │ TraceId           │ │
│ │                   │ │         │                   │         │ │                   │ │
│ │ ParentSpanId      │ │ Inject  │ X-B3-ParentSpanId │ Extract │ │ ParentSpanId      │ │
│ │                   ├─┼────────>│                   ├─────────┼>│                   │ │
│ │ SpanId            │ │         │ X-B3-SpanId       │         │ │ SpanId            │ │
│ │                   │ │         │                   │         │ │                   │ │
│ │ Sampling decision │ │         │ X-B3-Sampled      │         │ │ Sampling decision │ │
│ └───────────────────┘ │         └───────────────────┘         │ └───────────────────┘ │
│                       │                                       │                       │
└───────────────────────┘                                       └───────────────────────┘
```

跟踪标识符通常与***抽样决策***一起发送。但独立发送抽样决策是有效且常见的做法。下面是代理禁止跟踪端点的示例`/health`。该图显示接收器生成 NoOp 跟踪上下文以确保最小开销。

```
                                Server Tracer     
                              ┌───────────────────────┐
 Health check request         │                       │
┌───────────────────┐         │   TraceContext        │
│ GET /health       │ Extract │ ┌───────────────────┐ │
│ X-B3-Sampled: 0   ├─────────┼>│ NoOp              │ │
└───────────────────┘         │ └───────────────────┘ │
                              └───────────────────────┘
```

- #### Http编码

  B3有两种编码方式：Single Header（**单标头**编码）和Multiple Header（**多标头**编码）

  多标头编码-> 在跟踪上下文中为每个项目使用一个以***X-B3***为前缀的标头

  单标头编码-> 将上下文分隔为一个名为**b3**的单个条目

  1. ##### Multiple Header（**多标头**编码）

     B3比较常作为多个http头进行传播，遵循`X-B3-${name}]`的约定。

  2. ##### Single Header（**单标头**编码）

     b3将传播字段映射到以连字符分隔的字符串中：

     `b3={TraceId}-{SpanId}-{SamplingState}-{ParentSpanId}`（最后两个字段是可选的）

     例如，以下状态编码在多个标头中：

     ```
     X-B3-TraceId: 80f198ee56343ba864fe8b2a57d3eff7
     X-B3-ParentSpanId: 05e3ac9a4f6e3b90
     X-B3-SpanId: e457b5a2e4d86bd1
     X-B3-Sampled: 1
     ```

     成为一个`b3`标题：

     ```
     b3: 80f198ee56343ba864fe8b2a57d3eff7-e457b5a2e4d86bd1-1-05e3ac9a4f6e3b90
     ```

     

- ## W3C

W3C标准规范了http标头和值格式，对**如何在服务之间发送和修改上下文信息**做了标准化。

> 在分布式追踪遍历多个组件的过程中会遇到一致性问题：要求它在所有参与系统中具有唯一的标识符。而**追踪上下文传播**（Trace context propagation ）传递这个唯一的标识。
>
> 现在，追踪上下文传播会由每一个单独的追踪供应商提供。在这样一个多供应商的环境中会出现**互操作性问题**：
>
> 1. 不同的追踪供应商收集的tracing结果一个个独立开，无法关联，因为它们没有共享的唯一标识符；
> 2. 跨越不同的追踪供应商之间的边界追踪没法传播，因为没有一组统一商定的标识可以转发；
> 3. ........

对于上面这些问题，在单一平台提供商范围内不会产生什么很大的影响；但在如今高分布式的应用环境下，就很需要分布式追踪的**上下文传播标准**了，也就是***trace context***

- #### trace context 的HTTP标头格式

  所谓的传播上下文trace context分为两个单独的传播字段 -- traceparent和tracestate。

  它俩大概长这个样子：

  ```
  traceparent: 00-0af7651916cd43dd8448eb211c80319cb7ad6b7169203331-01
  ```

  ```
  tracestate: congo=t61rcWkgMzE
  ```

  来看一下这两个header的关联：

> 举个例子，假如系统中的client和server使用不同的追踪供应商：Congo和Rojo，在 Congo 系统中跟踪的client将以下标头添加到**出站** HTTP 请求中：
>
> ```
> traceparent: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01
> tracestate: congo=t61rcWkgMzE
> //tracestate 值 t61rcWkgMzE 是对父ID(b7ad6b7169203331)进行 Base64 编码的结果
> ```
>
> 在 Rojo 跟踪系统中跟踪的receive server，将它收到的 `tracestate` 带走，并在**左侧**添加一个新条目：
>
> ```
> traceparent: 00-0af7651916cd43dd8448eb211c80319c-00f067aa0ba902b7-01
> tracestate: rojo=00f067aa0ba902b7,congo=t61rcWkgMzE
> ```
>
> 注意到 Rojo 系统将 `traceparent` 的值重新用于在 `tracestate` 中的条目。这意味着它是一个**通用的跟踪系统**。否则， `tracestate` 条目是不透明的且可以是特定于供应商的。
>
> 如果下一个接收服务器使用 Congo，它会继承 Rojo 的 `tracestate` 并在前一个条目的左侧为父代添加一个新条目：
>
> ```
> traceparent: 00-0af7651916cd43dd8448eb211c80319c-b9c7c989f97918e1-01
> tracestate: congo=ucfJifl5GOE,rojo=00f067aa0ba902b7
> ```
>
> 最后会看到 `tracestate` 保留了 Rojo 的条目，除了被推到右边之外，最左边的位置让下一个服务器知道哪个跟踪系统对应于 `traceparent` 。

------



研究下了dubbo java的实现源码，发现在observability对tracing板块实现支持B3和W3C的具体代码在dubbo-spring-boot-observability目录下：

```java
// dubbo-spring-boot/dubbo-spring-boot-observability-starter/src/main/java/org/apache/dubbo/spring/boot/observability/config/DubboTracingProperties.java
```

`DubboTracingProperties`类中定义了：采样Sampling、Baggage和传播propagation等一些dubbo-tracing的追踪属性，后会在`OpenTelemetryAutoConfiguration`和`BraveAutoConfiguration`类中调用

类的框架整理如下：

![](https://github.com/yanyanran/pictures/blob/main/DubboTracingProperties.png?raw=true)