package registry

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// MyRegistry 注册中心
// 添加服务器并接收心跳以使其保持活动状态，同时返回所有活动服务器和删除失效服务器同步
type MyRegistry struct {
	timeout time.Duration
	mu      sync.Mutex
	servers map[string]*ServerItem
}

type ServerItem struct {
	Addr  string
	start time.Time
}

const (
	defaultPath    = "/_myrpc_/registry"
	defaultTimeout = time.Minute * 5 // 默认注册服务5min后作不可用状态
)

// New new一个具有超时设置的注册中心实例
func New(timeout time.Duration) *MyRegistry {
	return &MyRegistry{
		servers: make(map[string]*ServerItem),
		timeout: timeout,
	}
}

var DefaultMyRegister = New(defaultTimeout)

// putServer 添加服务实例
func (r *MyRegistry) putServer(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.servers[addr]
	if s == nil {
		r.servers[addr] = &ServerItem{
			Addr:  addr,
			start: time.Now(),
		}
	} else { // 服务存在=> 更新startTime
		s.start = time.Now()
	}
}

// aliveServers 返回可用的服务列表
func (r *MyRegistry) aliveServers() (alive []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for addr, s := range r.servers {
		if r.timeout == 0 || s.start.Add(r.timeout).After(time.Now()) { // 检查时间点timeout是否在now之后
			alive = append(alive, addr)
		} else { // 超时服务=> 删除
			delete(r.servers, addr)
		}
	}
	sort.Strings(alive)
	return alive
}

// ServeHTTP HTTP协议提供服务，运行在/_myrpc_/registry
func (r *MyRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET": // 返回所有可用的服务列表 通过自定义字段X-MyRPC-Servers承载
		w.Header().Set("X-MyRPC-Servers", strings.Join(r.aliveServers(), ","))
	case "POST": // 添加服务实例或发送心跳 通过自定义字段X-MyRPC-Server承载
		addr := req.Header.Get("X-MyRPC-Server")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.putServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (r *MyRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, r)
	log.Println("rpc注册中心路径：", registryPath)
}

func HandleHTTP() {
	DefaultMyRegister.HandleHTTP(defaultPath)
}

// Heartbeat 服务器注册/发送心跳的helper
func Heartbeat(registry, addr string, duration time.Duration) {
	if duration == 0 { // 默认心跳周期
		duration = defaultTimeout - time.Duration(1)*time.Minute
	}
	var err error
	err = sendHeartbeat(registry, addr)
	go func() {
		t := time.NewTicker(duration) // Ticker周期触发定时的定时器，会按一个时间间隔往channel发送系统当前时间
		for err == nil {
			<-t.C
			err = sendHeartbeat(registry, addr)
			t.Stop()
		}
	}()
}

func sendHeartbeat(registry, addr string) error {
	log.Println(addr, "发送心跳到注册中心", registry)
	httpClient := &http.Client{}
	req, _ := http.NewRequest("POST", registry, nil)
	req.Header.Set("X-MyRPC-Server", addr)
	if _, err := httpClient.Do(req); err != nil { // Do发送HTTP请求并返回HTTP响应
		log.Println("rpc服务器：心跳错误：", err)
		return err
	}
	return nil
}
