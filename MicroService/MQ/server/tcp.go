package server

import (
	"context"
	"log"
	"net"
)

// TcpServer 监听客户端tcp连接
func TcpServer(ctx context.Context, addr, port string) {
	fqAddress := addr + ":" + port
	listener, err := net.Listen("tcp", fqAddress)
	if err != nil {
		panic("tcp listen(" + fqAddress + ") failed")
	}
	log.Printf("listening for clients on %s", fqAddress)

	for {
		select {
		case <-ctx.Done():
			return
		default: // 监听到tcp连接
			conn, err := listener.Accept()
			if err != nil {
				panic("accept failed: " + err.Error())
			}
			client := NewClient(conn, conn.RemoteAddr().String()) // new一个client处理
			// 传入context方便统一退出管理
			// TODO：对client和protocol也要加上对此ctx的监听代码
			go client.Handle(ctx)
		}
	}
}
