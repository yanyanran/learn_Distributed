package xclient

import (
	"context"
	"io"
	. "myrpc"
	"reflect"
	"sync"
)

type XClient struct {
	dis     Discovery
	mode    SelectMode
	opt     *Option
	mu      sync.Mutex
	clients map[string]*Client // client缓存
}

func NewXClient(dis Discovery, mode SelectMode, opt *Option) *XClient {
	return &XClient{
		dis:     dis,
		mode:    mode,
		opt:     opt,
		clients: make(map[string]*Client),
	}
}

var _ io.Closer = (*XClient)(nil)

func (X *XClient) Close() error {
	X.mu.Lock()
	defer X.mu.Unlock()
	for rpcAddrKey, client := range X.clients {
		// ignore error
		client.Close()
		delete(X.clients, rpcAddrKey)
	}
	return nil
}

// dial 复用Client
func (X *XClient) dial(rpcAddr string) (*Client, error) {
	X.mu.Lock()
	defer X.mu.Unlock()
	client, ok := X.clients[rpcAddr] // Q: 检查X.clients里是否有缓存的Client
	if ok && !client.IsAvailable() { // Q-> 有，Q2：检查是否处于可用状态
		client.Close()
		delete(X.clients, rpcAddr) // Q2-> 不可用，从缓存中删除
		client = nil
	}
	if client == nil { // Q-> 无缓存 创建新的返回新的
		var err error
		client, err = XDial(rpcAddr, X.opt)
		if err != nil {
			return nil, err
		}
		X.clients[rpcAddr] = client
	}
	return client, nil // Q2-> 可用，直接返回缓存的client
}

func (X *XClient) call(rpcAddr string, ctx context.Context, serviceMethod string, args, reply interface{}) error {
	client, err := X.dial(rpcAddr)
	if err != nil {
		return err
	}
	return client.Call(ctx, serviceMethod, args, reply)
}

// Call 调用命名函数，等待它完成并返回它的错误状态。 X将选择合适的服务器
func (X *XClient) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	rpcAddr, err := X.dis.Get(X.mode) // 选择一个服务
	if err != nil {
		return err
	}
	return X.call(rpcAddr, ctx, serviceMethod, args, reply)
}

// Broadcast 为discovery中注册的每个服务器调用命名函数，将请求广播到所有的服务实例
func (X *XClient) Broadcast(ctx context.Context, serverMethod string, args, reply interface{}) error {
	servers, err := X.dis.GetAll()
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var e error
	replyDone := reply == nil              // 如果reply为空，则不需要设置
	ctx, cancel := context.WithCancel(ctx) // 确保有错误发生时，快速失败
	for _, rpcAddr := range servers {
		wg.Add(1)
		go func(rpcAddr string) { // goroutine并发
			defer wg.Done()
			var clonedReply interface{}
			if reply != nil {
				clonedReply = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface()
			}
			err := X.call(rpcAddr, ctx, serverMethod, args, clonedReply)
			mu.Lock() // 加锁 防止多个协程同时写e
			if err != nil && e == nil {
				e = err
				cancel() // 如果任何一个call失败，就取消未完成的call
			}
			if err == nil && !replyDone {
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(clonedReply).Elem())
				replyDone = true
			}
			mu.Unlock()
		}(rpcAddr)
	}
	wg.Wait()
	return e
}
