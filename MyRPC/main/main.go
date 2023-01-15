package main

import (
	"encoding/json"
	"fmt"
	"log"
	MyRPC "myrpc"
	"myrpc/codec"
	"net"
	"time"
)

func startServer(addr chan string) {
	lis, err := net.Listen("tcp", ":9999")
	if err != nil {
		log.Fatal("网络错误：", err)
	}
	log.Println("start rpcServer on", lis.Addr()) // Addr返回监听器lis的网络地址
	addr <- lis.Addr().String()                   // string形式的地址
	MyRPC.Accept(lis)                             // run server
}

func main() {
	addr := make(chan string) // 信道确保server端口监听成功 client再发起请求
	go startServer(addr)

	// 一个简单的MyRPC客户端
	conn, _ := net.Dial("tcp", <-addr) // Dial(网络协议名，IP地址/域名)创建网络连接
	defer func() {
		conn.Close()
	}()
	time.Sleep(time.Second)
	// 1、发送Option进行协议交换
	json.NewEncoder(conn).Encode(MyRPC.DefaultOption)
	cc := codec.NewGobCodec(conn)

	for i := 0; i < 5; i++ {
		h := &codec.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		cc.Write(h, fmt.Sprintf("MyRPC 请求 %d", h.Seq)) //  2、发送消息头+消息体
		cc.ReadHeader(h)
		var reply string
		cc.ReadBody(&reply) // 3、解析server的响应reply，打印
		log.Println("回复:", reply)
	}
}
