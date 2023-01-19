package main

import (
	"context"
	"log"
	MyRPC "myrpc"
	"myrpc/xclient"
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

// Sleep 验证XClient的超时机制能否正常运作
func (f Foo) Sleep(args Args, reply *int) error {
	time.Sleep(time.Second * time.Duration(args.Num1))
	*reply = args.Num1 + args.Num2
	return nil
}

func startServer(addr chan string) {
	var foo Foo
	/*	if err := MyRPC.Register(&foo); err != nil {
			log.Fatal("register发生错误：", err)
		}
		lis, err := net.Listen("tcp", ":9999")
		if err != nil {
			log.Fatal("网络错误：", err)
		}
		log.Println("start rpcServer on", lis.Addr()) // Addr返回监听器lis的网络地址
		addr <- lis.Addr().String()                   // string形式的地址
		MyRPC.Accept(lis)                             // run server*/
	lis, _ := net.Listen("tcp", ":0")
	server := MyRPC.NewServer()
	server.Register(&foo)
	//MyRPC.HandleHTTP()
	addr <- lis.Addr().String()
	server.Accept(lis)
	//http.Serve(lis, nil)
}

// foo 便于在Call或Broadcast之后统一打印成功/失败的日志
func foo(xc *xclient.XClient, ctx context.Context, typ, serviceMethod string, args *Args) {
	var reply int
	var err error
	switch typ {
	case "call":
		err = xc.Call(ctx, serviceMethod, args, &reply)
	case "broadcast":
		err = xc.Broadcast(ctx, serviceMethod, args, &reply)
	}
	if err != nil {
		log.Printf("%s %s error: %v", typ, serviceMethod, err)
	} else {
		log.Printf("%s %s success: %d + %d = %d", typ, serviceMethod, args.Num1, args.Num2, reply)
	}
}

func call(addr1, addr2 string) {
	d := xclient.NewMultiServerDiscovery([]string{"tcp@" + addr1, "tcp@" + addr2})
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() { _ = xc.Close() }()
	// 发送请求和接收响应
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(xc, context.Background(), "call", "Foo.Sum", &Args{Num1: i, Num2: i * i})
		}(i)
	}
	wg.Wait()
}

func broadcast(addr1, addr2 string) {
	d := xclient.NewMultiServerDiscovery([]string{"tcp@" + addr1, "tcp@" + addr2})
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() { _ = xc.Close() }()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(xc, context.Background(), "broadcast", "Foo.Sum", &Args{Num1: i, Num2: i * i})
			// 预期2-5超时
			ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
			foo(xc, ctx, "broadcast", "Foo.Sleep", &Args{Num1: i, Num2: i * i})
		}(i)
	}
	wg.Wait()
}

func main() {
	log.SetFlags(0)
	ch1 := make(chan string)
	ch2 := make(chan string)
	go startServer(ch1) // 启动两台服务器
	go startServer(ch2)
	addr1 := <-ch1
	addr2 := <-ch2
	time.Sleep(time.Second)
	call(addr1, addr2)
	broadcast(addr1, addr2)
}

func callOld(addrCh chan string) {
	client, _ := MyRPC.DialHTTP("tcp", <-addrCh)
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
			args := &Args{Num1: i, Num2: i * i}
			var reply int
			if err := client.Call(context.Background(), "Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Printf("%d + %d = %d", args.Num1, args.Num2, reply)
		}(i)
	}
	wg.Wait()
}

func mainOld() {
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

func mainHTTP() {
	log.SetFlags(0)
	ch := make(chan string)
	//go call(ch)
	startServer(ch)
}
