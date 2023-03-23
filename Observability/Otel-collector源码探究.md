# Otel-collector源码探究

*不得不看一下Otel的源码了.......*

先从Opentelementry中最核心的collector部分下手看看，这部分由三个组件构成：

- `Receivers`: 接受数据。负责按照对应的协议格式监听接收遥测数据，并把数据转给一个或者多个processor
- `Processors`: 处理数据。把数据传给下一个processor或者传给一个或多个exporter
- `Exporters`: 把数据暴露出去

使用一种**pipeline**（一套数据的流入、处理、流出过程）把它们三个聚合起来，还有一些 `extensions` 用来拓展 collector

![](https://github.com/yanyanran/pictures/blob/main/Otel%20pipeline.png?raw=true)

Collector可以接收otlp、zipkin、jaeger等任意格式的数据，然后以otlp、zipkin、jaeger等任意格式的数据转发出去。这一切只取决于你需要输入或输出的格式是否有对应的receiver和exporter实现。

------

先从main函数下手。

身为一个处理遥测数据的框架，由于用户上报的操作系统环境在实际情况中不确定（Otel只配置了win系统下的启动函数在main_windows.go，其他系统都是main_other.go。这两个文件中的run方法将用在main中），所以看到Otel允许用户自己编写对应的插件从而配置到服务框架中去，所以它区别于以往的暴露一个可继承的服务实例，Otel暴露了一个命令实例command：

```go
// main/main.go
// use in other system
func runInteractive(params otelcol.CollectorSettings) error {
	cmd := otelcol.NewCommand(params)  // 新建服务
	if err := cmd.Execute(); err != nil {  // 启动服务
		log.Fatalf("collector server run finished with error: %v", err)
	}

	return nil
}
```

从架构层面来说，Collecter有两种模式：

> 1. 把Collecter部署在相同的主机内，应用获取遥测数据之后，直接通过回环网络往Collecter传递，这种叫**Agent模式**；（不解耦）
> 2. 将Collecter封装成一个独立的中间件，应用把采集到的数据往这个中间件里传递，这种叫**Geteway模式**（解耦）

这两种模式可以一起用，只需数据出口的数据协议格式跟数据入口的数据协议格式保持一致。