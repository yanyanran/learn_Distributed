package MyRPC

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"myrpc/codec"
	"net"
	"reflect"
	"sync"
)

const MagicNum = 0x3bef5c

type Server struct{}            // an rpc server
var DefaultServer = NewServer() // *Server的默认实例

// Option 客户端可选择不同的编解码器来编码消息body
type Option struct {
	MagicNum  int        // MagicNum 标记这是一个myRPC请求
	CodecType codec.Type // choose
}

var DefaultOption = &Option{ // 默认选项
	MagicNum:  MagicNum,
	CodecType: codec.GobType,
}

func NewServer() *Server {
	return &Server{}
}

// Accept 默认的Server实例，为了用户方便
func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}

// Accept 在listener上接受连接 并为每个传入连接提供请求
func (server *Server) Accept(lis net.Listener) {
	for { // 循环等待socket连接建立
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc服务器：accept错误：", err)
			return
		}
		go server.ServerConn(conn) // 子协程处理
	}
}

// ServerConn 扣通信过程。在单个连接上运行server(阻塞连接直到客户端挂断
func (server *Server) ServerConn(conn io.ReadWriteCloser) {
	defer func() {
		conn.Close()
	}()
	var opt Option
	// json.NewDecoder反序列化得到 Option 实例
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc服务器：option错误：", err)
		return
	}
	// check out MagicNumber
	if opt.MagicNum != MagicNum {
		log.Printf("rpc服务器：无效的magic number %x\n", opt.MagicNum)
		return
	}
	// check out CodecType
	f := codec.NewCodecFuncMap[opt.CodecType] // f(value)->NewCodeFunc类型
	if f == nil {
		log.Printf("rpc服务器：无效的codec type %s\n", opt.CodecType)
		return
	}
	server.serverCodec(f(conn))
}

var invalidRequest = struct{}{} // 发生错误时响应argv的占位符

// serverCodec 对请求读、回复、处理
func (server *Server) serverCodec(cc codec.Codec) {
	send := new(sync.Mutex) // 互斥锁 确保发送完整的响应(避免多个回复交织在一起客户端无法正确解析)
	/* WaitGroup 对象内部有一个计数器，最初从0开始，三个方法：Add(), Done(), Wait() 用来控制计数器数量
	   Add(n) 计数器设为n
	   Done() 每次计数器-1
	   wait() 阻塞代码运行，直到计数器减为0 */
	wait := new(sync.WaitGroup) // 等所有请求得到处理(原sleep做法)
	for {
		req, err := server.readRequest(cc) // 读
		if err != nil {
			if req == nil {
				break // nil关闭连接
			}
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, send) // 回复（逐个发）
			continue
		}
		wait.Add(1)
		go server.handleRequest(cc, req, send, wait) // 并发处理
	}
	wait.Wait()
	cc.Close()
}

// request 存储call的所有信息
type request struct {
	h      *codec.Header
	argv   reflect.Value // 反射值对象
	replyv reflect.Value
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc服务器：读取header错误：", err)
		}
		return nil, err
	}
	return &h, nil
}

// readRequest 读取请求
func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	// TODO: 现在我们不确定请求argv的类型，先假设是字符串
	req.argv = reflect.New(reflect.TypeOf(""))
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc服务器：读取argv错误：", err)
	}
	return req, nil
}

// sendResponse 回复请求
func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, send *sync.Mutex) {
	send.Lock()
	defer send.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc服务器：写入响应错误：", err)
	}
}

// handleRequest 处理请求
func (server *Server) handleRequest(cc codec.Codec, req *request, send *sync.Mutex, wait *sync.WaitGroup) {
	// TODO 应调用已注册的rpc方法以获得正确的replyv
	// 简单的，目前--只需打印argv并发送hello消息
	defer wait.Done()
	log.Println(req.h, req.argv.Elem())                                      // reflect.Elem()通过反射获取指针指向的元素类型
	req.replyv = reflect.ValueOf(fmt.Sprintf("myrpc respone %d", req.h.Seq)) /// Seq客户端请求序列号
	server.sendResponse(cc, req.h, req.replyv.Interface(), send)
}