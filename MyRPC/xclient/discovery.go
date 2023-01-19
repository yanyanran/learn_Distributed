package xclient

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

// SelectMode 不同的负载均衡策略
type SelectMode int

const (
	RandomSelect     SelectMode = iota // 随机选择
	RoundRobinSelect                   // 轮询选择
)

type Discovery interface {
	Refresh() error                      // 从远程注册中心更新服务列表
	Update(servers []string) error       // 手动更新服务列表
	Get(mode SelectMode) (string, error) // 根据负载均衡策略，选择一个服务实例
	GetAll() ([]string, error)           // 返回所有的服务实例
}

// MultiServersDiscovery 发现没有注册中心的多服务器 用户显式提供服务器地址
type MultiServersDiscovery struct {
	mu      sync.RWMutex // 读写锁=> 基于Mutex实现
	ran     *rand.Rand   // 产生随机数的实例
	servers []string
	index   int // 记录轮询算法已经轮询到的位置(避免从0开始
}

// NewMultiServerDiscovery 创建MultiServersDiscovery实例
func NewMultiServerDiscovery(servers []string) *MultiServersDiscovery {
	d := &MultiServersDiscovery{
		ran:     rand.New(rand.NewSource(time.Now().UnixNano())),
		servers: servers,
	}
	d.index = d.ran.Intn(math.MinInt32 - 1)
	return d
}

var _ Discovery = (*MultiServersDiscovery)(nil) // 实现Discovery接口

func (d *MultiServersDiscovery) Refresh() error {
	return nil
}

func (d *MultiServersDiscovery) Update(servers []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = servers
	return nil
}

func (d *MultiServersDiscovery) Get(mode SelectMode) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.servers) == 0 {
		return "", errors.New("rpc discovery: 没有可用的服务器")
	}
	switch mode {
	case RandomSelect:
		return d.servers[d.ran.Intn(len(d.servers))], nil
	case RoundRobinSelect:
		s := d.servers[d.index%len(d.servers)] // 服务器可更新，因此模式len(d.servers)可以确保安全
		d.index = (d.index + 1) % len(d.servers)
		return s, nil
	default:
		return "", errors.New("rpc discovery: 不支持的Select模式")
	}
}

func (d *MultiServersDiscovery) GetAll() ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	servers := make([]string, len(d.servers), len(d.servers)) // 返回d.server的副本
	copy(servers, d.servers)
	return servers, nil
}
