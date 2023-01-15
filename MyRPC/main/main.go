package main

import (
	MyRPC "myrpc"
	"net"
)

func main() {
	lis, _ := net.Listen("tcp", ":9999")
	MyRPC.Accept(lis) // run server
}
