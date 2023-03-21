# 追踪标准之--B3和W3C

1. > *最近接的Dubbo-go项目，负责tracing模块，第一阶段是让tracing支持B3和W3C追踪标准*。

- ## B3

B3最初由Zipkin项目开发，目前成为OpenTracing标准之一。B3标准使用HTTP头传递跟踪信息，以确保分布式系统中所有服务都可以访问该信息。B3使用以下三个HTTP头：X-B3-TraceId、X-B3-SpanId、X-B3-ParentSpanId（以X-B3作为开头，用来跟踪跨服务边界的TraceContext（上下文传播）

- ##### 整体流程

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

跟踪标识符通常与***抽样决策***（？）一起发送。但独立发送抽样决策是有效且常见的做法。下面是代理禁止跟踪端点的示例`/health`。该图显示接收器生成 NoOp 跟踪上下文以确保最小开销。

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

