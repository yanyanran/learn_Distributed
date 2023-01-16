package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	MyRPC "myrpc"
	"myrpc/codec"
	"net"
	"sync"
)

// Call 一次RPC调用所需信息的封装（客户端发送到server 被拆分存在request->head+body）
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

// receive 接收响应
func (client *Client) receive() {
	var err error
	for err == nil { // for一直轮询调用直到err!=nil
		var h codec.Header
		if err = client.cc.ReadHeader(&h); err != nil {
			break
		}
		call := client.removeCall(h.Seq)

		switch {
		case call == nil: // call不存在 通常意味着写部分失败且call已被删除
			err = client.cc.ReadBody(nil)
		case h.Error != "": // call存在但服务端处理出错（h.Error不为空
			call.Error = fmt.Errorf(h.Error)
			err = client.cc.ReadBody(nil)
			call.done()
		default: // call存在 服务端也正常 ->从body中读取Reply值
			err = client.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}
	// 发生错误
	client.terminateCalls(err)
}

func NewClient(conn net.Conn, opt *MyRPC.Option) (*Client, error) {
	f := codec.NewCodecFuncMap[opt.CodecType] // 编解码func
	if f == nil {
		err := fmt.Errorf("无效的编解码器类型%s", opt.CodecType)
		log.Println("rpc客户端：编解码器错误：", err)
		return nil, err
	}
	// 发送option给server
	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("rpc客户端：option错误：", err)
		conn.Close()
		return nil, err
	}
	return NewClientCodec(f(conn), opt), nil // 协商好消息编解码方式后
}

func NewClientCodec(cc codec.Codec, opt *MyRPC.Option) *Client {
	client := &Client{
		seq:  1, // seq以1开头，0表示无效call
		cc:   cc,
		opt:  opt,
		pend: make(map[uint64]*Call),
	}
	go client.receive() // 创建子协程接收响应
	return client
}

// parseOptions 解析Option
func parseOptions(opts ...*MyRPC.Option) (*MyRPC.Option, error) {
	// 如果opts为nil或传递nil作为参数 --> 使用默认的
	if len(opts) == 0 || opts[0] == nil {
		return MyRPC.DefaultOption, nil
	}
	if len(opts) != 1 {
		return nil, errors.New("option数大于1")
	}
	opt := opts[0]
	opt.MagicNum = MyRPC.DefaultOption.MagicNum
	if opt.CodecType == "" {
		opt.CodecType = MyRPC.DefaultOption.CodecType
	}
	return opt, nil
}

// Dial 客户端创建连接
func Dial(network, address string, opts ...*MyRPC.Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}
	conn, err := net.Dial(network, address) // func Dial(net, addr string) (Conn, error)创建网络连接
	if err != nil {
		return nil, err
	}
	// 如果客户端为空，则关闭连接
	defer func() {
		if client == nil {
			conn.Close()
		}
	}()
	return NewClient(conn, opt)
}

// Send 客户端发送请求
func (client *Client) Send(call *Call) {
	client.send.Lock()
	defer client.send.Unlock() // 确保client发送完整的请求

	seq, err := client.registerCall(call) // 注册这个call
	if err != nil {
		call.Error = err
		call.done()
		return
	}
	// 准备请求head
	client.header.ServiceMethod = call.ServiceMethod
	client.header.Seq = seq
	client.header.Error = ""

	// 编码及发送请求
	if err := client.cc.Write(&client.header, call.Args); err != nil {
		call := client.removeCall(seq)
		if call != nil { // call可能为nil（Write部分失败），客户端已收到response并处理
			call.Error = err
			call.done()
		}
	}
}

// Go 异步调用函数。返回表示调用的Call
func (client *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Panic("rpc客户端：done通道未缓冲")
	}
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}
	client.Send(call)
	return call
}

// Call 同步调用函数。阻塞call.Done，等待response到达后返回其错误状态
func (client *Client) Call(serviceMethod string, args, reply interface{}) error {
	// 读管道->管道为空会阻塞,直到server向管道发送response
	call := <-client.Go(serviceMethod, args, reply, make(chan *Call, 1)).Done
	return call.Error
}
