package test

import (
	"fmt"
	"log"
	"net/http"
	"testing"
)

type MyHttpServer struct {
	content string
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Header)
	fmt.Println("URL：", r.URL)
	fmt.Println("正在访问接口...")
}

// 实现Handler接口
func (s MyHttpServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	fmt.Println("正在访问...", s.content, "Header：", req.Header)
	w.Write([]byte("write..."))
}

func TestHttp(t *testing.T) {
	// 设置访问路由
	http.HandleFunc("/test", testHandler)               // HandleFunc处理器-> 自定义
	http.Handle("/t", MyHttpServer{content: "content"}) // Handle处理器-> 实现接口ServeHttp
	// 创建监听端口
	err := http.ListenAndServe("127.0.0.1:9090", nil)
	if err != nil {
		log.Fatal("ListenAndServe:	", err)
	}
}
