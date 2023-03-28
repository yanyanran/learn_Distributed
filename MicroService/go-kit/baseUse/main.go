package main

import (
	"MyRPC/MicroService/go-kit/baseUse/end_point"
	"MyRPC/MicroService/go-kit/baseUse/service"
	"MyRPC/MicroService/go-kit/baseUse/transport"
	"fmt"
	"net/http"
)

func main() {
	server := service.NewService()
	endpoints := end_point.NewEndPointServer(server)
	httpHandler := transport.NewHttpHandler(endpoints)
	fmt.Println("server run 0.0.0.0:8888")
	_ = http.ListenAndServe("0.0.0.0:8888", httpHandler)
}
