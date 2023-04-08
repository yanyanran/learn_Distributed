package main

import (
	"MQ/message"
	"MQ/server"
	"MQ/util"
	"context"
	"flag"
	"os"
	"os/signal"
	"strconv"
)

// 启动服务

// 绑定
var bindAddress = flag.String("address", "", "address to bind to")
var webPort = flag.Int("web-port", 5150, "port to listen on for HTTP connections")
var tcpPort = flag.Int("tcp-port", 5151, "port to listen on for TCP connections")
var memQueueSize = flag.Int("mem-queue-size", 10000, "number of messages to keep in memory (per topic)")

func main() {
	flag.Parse() // 解析命令行参数

	endChan := make(chan struct{})
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt) // signalChan监听中断信号

	ctx, fn := context.WithCancel(context.Background())
	go func() {
		<-signalChan
		fn()
	}()

	go message.TopicFactory(ctx, *memQueueSize) // new topic
	go util.UuidFactory(ctx)
	go server.TcpServer(ctx, *bindAddress, strconv.Itoa(*tcpPort))
	server.HttpServer(ctx, *bindAddress, strconv.Itoa(*webPort), endChan)

	for _, topic := range message.TopicMap {
		topic.Close()
	}
	//<-endChan
}
