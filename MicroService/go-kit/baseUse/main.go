package main

import (
	"go-kit/baseUse/end_point"
	"go-kit/baseUse/service"
	"go-kit/baseUse/transport"
	"go-kit/baseUse/utils"
	"log"
	"net/http"
)

func main() {
	utils.NewLoggerServer()
	server := service.NewService(utils.GetLogger())
	endpoints := end_point.NewEndPointServer(server, utils.GetLogger())
	httpHandler := transport.NewHttpHandler(endpoints, utils.GetLogger())
	log.Printf("server run 172.0.0.1:8888")
	_ = http.ListenAndServe("127.0.0.1:8888", httpHandler)
}
