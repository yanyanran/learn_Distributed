package client

import (
	"errors"
	"io"
	MyRPC "myrpc"
	"myrpc/codec"
	"sync"
)

// Call 一次RPC调用所需信息的封装
type Call struct {
	Seq           uint64
	ServiceMethod string
	Args          interface{} // func参数
	Reply         interface{} // func回复
	Error         error
	Done          chan *Call // 支持异步调用
}

// done 调用结束时会调用call.done()通知调用方
func (call *Call) done() {
	call.Done <- call
}

// Client 可能有多个未完成的Call与单个Client关联，且一个Client可能同时被多个 goroutines 使用
type Client struct {
	cc       codec.Codec
	opt      *MyRPC.Option
	send     sync.Mutex
	header   codec.Header
	mu       sync.Mutex
	seq      uint64           // seq 给发送的请求编号
	pend     map[uint64]*Call // pend 存储未处理完的请求 <K:编号 V:Call实例>
	closing  bool             // 主动
	shutdown bool             // 被动
}

var _ io.Closer = (*Client)(nil)
var ErrShutdown = errors.New("连接已关闭")

// Close 关闭连接
func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing {
		return ErrShutdown
	}
	client.closing = true
	return client.cc.Close()
}

// IsAvailable client工作返回true
func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return !client.shutdown && !client.closing // check out closing and shutdown
}

// registerCall 登记Call:将call添加到 client.pend map 中，并更新 client.seq
func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing || client.shutdown {
		return 0, ErrShutdown
	}
	call.Seq = client.seq
	client.pend[call.Seq] = call
	client.seq++
	return call.Seq, nil
}

// removeCall 根据seq，从 client.pend mao 中移除对应的call并返回
func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()
	call := client.pend[seq]
	delete(client.pend, seq)
	return call
}

// terminateCalls server或client发生错误时调用
func (client *Client) terminateCalls(err error) {
	client.send.Lock()
	defer client.send.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()
	// shutdown设为true，且将错误信息通知给所有pend状态的call
	client.shutdown = true
	for _, call := range client.pend {
		call.Error = err
		call.done()
	}
}
