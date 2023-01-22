package main

import (
	. "MyCache"
	"flag"
	"fmt"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createGroup() *Group {
	return NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] 搜索key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s不存在", key)
		}))
}

// startCacheServer 启动缓存服务器（本测试三个port
func startCacheServer(addr string, addrs []string, gee *Group) {
	peers := NewHTTPPool(addr)
	peers.Set(addrs...)
	gee.RegisterPeers(peers)
	log.Println("mycache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

// startAPIServer 启动一个API服务（端口9999）与用户交互（用户感知
func startAPIServer(apiAddr string, myGroup *Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := myGroup.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())

		}))
	log.Println("API服务器正运行在", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "MyCache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999" // 用户交互端口
	addrMap := map[int]string{         // 三个HTTP服务器端口（用户不感知
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	myGroup := createGroup()
	if api {
		go startAPIServer(apiAddr, myGroup)
	}
	startCacheServer(addrMap[port], []string(addrs), myGroup)
}
