package MyCache

import (
	"fmt"
	"log"
	"sync"
)

// mycache主流程：与外部交互、控制缓存存储和获取

type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc 使用函数实现Getter
type GetterFunc func(key string) ([]byte, error)

// Get 缓存不存在=> 调用回调函数得到源数据
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// Group 缓存的命名空间
type Group struct {
	name      string
	getter    Getter // 缓存未hit时获取源数据的回调
	mainCache cache  // 一开始实现的并发缓存
	peers     PeerPicker
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get 从缓存中获取key的值
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("需要key")
	}
	if v, ok := g.mainCache.get(key); ok { // 流程⑴ 从mainCache中查找缓存，存在则返回缓存值
		log.Println("[MyCache] hit")
		return v, nil
	}
	defer fmt.Printf("%s's cache does not exist and new cache is now created", key)
	return g.load(key) // 流程⑶ 缓存不存在
}

// RegisterPeers 注册PeerPicker以选择远程peer注入Group
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker调用了多次")
	}
	g.peers = peers
}

// 设计预留:分布式场景下，load会先从远程节点获取getFromPeer，失败了再回退到getLocally
func (g *Group) load(key string) (value ByteView, err error) {
	if g.peers != nil {
		if peer, ok := g.peers.PickPeer(key); ok {
			if value, err := g.getFromPeer(peer, key); err == nil {
				return value, nil
			}
			log.Println("[MyCache] 无法从peer获取", err)
		}
	}
	return g.getLocally(key)
}

// getFromPeer 访问远程peer获取缓存值
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key) // 使用用户回调函数Get获取源数据
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value) // 将源数据添加到缓存mainCache中
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
