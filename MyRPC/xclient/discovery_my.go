package xclient

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type MyRegistryDiscovery struct {
	*MultiServersDiscovery               // 嵌套服务发现
	registry               string        // 注册中心地址
	timeout                time.Duration // 服务列表过期时间
	lastUpdate             time.Time     // 从注册中心更新服务列表时间（default 10s
}

const defaultUpdateTimeout = time.Second * 10

func NewMyRegistryDiscovery(registerAddr string, timeout time.Duration) *MyRegistryDiscovery {
	if timeout == 0 {
		timeout = defaultUpdateTimeout
	}
	d := &MyRegistryDiscovery{
		MultiServersDiscovery: NewMultiServerDiscovery(make([]string, 0)),
		registry:              registerAddr,
		timeout:               timeout,
	}
	return d
}

func (d *MyRegistryDiscovery) Update(servers []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = servers
	d.lastUpdate = time.Now()
	return nil
}

// Refresh 超时重新获取
func (d *MyRegistryDiscovery) Refresh() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.lastUpdate.Add(d.timeout).After(time.Now()) { // 没超时
		return nil
	}
	log.Println("rpc注册中心：从注册中心" + d.registry + "刷新服务器")
	resp, err := http.Get(d.registry) // send GET
	if err != nil {
		log.Println("rpc注册中心刷新错误：", err)
		return err
	}
	servers := strings.Split(resp.Header.Get("X-MyRPC-Servers"), ",") // reget
	d.servers = make([]string, 0, len(servers))
	for _, server := range servers {
		if strings.TrimSpace(server) != "" {
			d.servers = append(d.servers, strings.TrimSpace(server))
		}
	}
	d.lastUpdate = time.Now()
	return nil
}

func (d *MyRegistryDiscovery) Get(mode SelectMode) (string, error) {
	if err := d.Refresh(); err != nil { // 先Refresh确保服务列表没有过期
		return "", err
	}
	return d.MultiServersDiscovery.Get(mode)
}

func (d *MyRegistryDiscovery) GetAll() ([]string, error) {
	if err := d.Refresh(); err != nil { // 先Refresh确保服务列表没有过期
		return nil, err
	}
	return d.MultiServersDiscovery.GetAll()
}
