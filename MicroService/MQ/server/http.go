package server

import (
	"MQ/message"
	"MQ/protocol"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type ReqParams struct {
	params url.Values // url参数集合
	body   []byte
}

func NewReqParams(req *http.Request) (*ReqParams, error) {
	reqParams, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	return &ReqParams{reqParams, data}, nil
}

func (r *ReqParams) Query(key string) (string, error) {
	keyData := r.params[key]
	if len(keyData) == 0 {
		return "", errors.New("key not in query params")
	}
	return keyData[0], nil
}

// 测试连接
func pingHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Length", "2")
	io.WriteString(w, "OK")
}

// 写入消息
func putHandler(w http.ResponseWriter, req *http.Request) {
	reqParams, err := NewReqParams(req)
	if err != nil {
		log.Printf("HTTP: error - %s", err.Error())
		return
	}
	topicName, err := reqParams.Query("topic") // 获取topic
	if err != nil {
		log.Printf("HTTP: error - %s", err.Error())
		return
	}

	conn := &FakeConn{} // 假的client
	client := NewClient(conn, "HTTP")
	proto := &protocol.Protocol{}
	// 让fake client向协议发送PUB，由协议和topic交互
	resp, err := proto.Execute(client, "PUB", topicName, string(reqParams.body))
	if err != nil {
		log.Printf("HTTP: error - %s", err.Error())
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(resp)))
	w.Write(resp)
}

// 查看所有topic
func statsHandler(w http.ResponseWriter, req *http.Request) {
	for topicName, _ := range message.TopicMap {
		io.WriteString(w, fmt.Sprintf("%s\n", topicName))
	}
}

// HttpServer 启动一个Http server
func HttpServer(ctx context.Context, address string, port string, endChan chan struct{}) {
	http.HandleFunc("/ping", pingHandler)
	http.HandleFunc("/put", putHandler)
	http.HandleFunc("/stats", statsHandler)

	fqAddress := address + ":" + port
	httpServer := http.Server{
		Addr: fqAddress,
	}

	go func() {
		log.Printf("listening for http requests on %s", fqAddress)
		err := http.ListenAndServe(fqAddress, nil)
		if err != nil {
			log.Fatal("http.ListenAndServe:", err)
		}
	}()

	<-ctx.Done() // 监听到退出信号后 生成一个带超时时间的context
	log.Printf("HTTP server on %s is shutdowning...", fqAddress)
	timeoutCtx, fn := context.WithTimeout(context.Background(), 10*time.Second) // there
	defer fn()
	if err := httpServer.Shutdown(timeoutCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	close(endChan)
}
