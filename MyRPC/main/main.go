package main

import (
	"context"
	"log"
	MyRPC "myrpc"
	"net"
	"sync"
	"time"
)

type Foo int

type Args struct {
	Num1, Num2 int
}

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func startServer(addr chan string) {
	var foo Foo
	if err := MyRPC.Register(&foo); err != nil {
		log.Fatal("register发生错误：", err)
	}
	lis, err := net.Listen("tcp", ":9999")
	if err != nil {
		log.Fatal("网络错误：", err)
	}
	log.Println("start rpcServer on", lis.Addr()) // Addr返回监听器lis的网络地址
	addr <- lis.Addr().String()                   // string形式的地址
	MyRPC.Accept(lis)                             // run server
}

func main() {
	log.SetFlags(0)
	addr := make(chan string) // 信道确保server端口监听成功 client再发起请求
	go startServer(addr)

	// 一个简单的MyRPC客户端
	/*	conn, _ := net.Dial("tcp", <-addr) // Dial(网络协议名，IP地址/域名)创建网络连接
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
		}*/

	// 使用client.Call并发5个RPC同步调用
	client, _ := MyRPC.Dial("tcp", <-addr)
	defer func() {
		client.Close()
	}()
	time.Sleep(time.Second)

	// 发送请求和接收响应
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			/*			args := fmt.Sprintf("MyRPC req %d", i)
						var reply string*/
			args := &Args{Num1: i, Num2: i * i}
			var reply int
			ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
			if err := client.Call(ctx, "Foo.Sum", args, &reply); err != nil {
				log.Fatal("调用Foo.Sum错误：", err)
			}
			//log.Println("回复: ", reply)
			log.Printf("%d + %d = %d", args.Num1, args.Num2, reply)
		}(i)
	}
	wg.Wait()
}
