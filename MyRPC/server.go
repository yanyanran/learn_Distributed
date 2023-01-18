package MyRPC

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"myrpc/codec"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
)

const MagicNum = 0x3bef5c

type Server struct { // an rpc server
	serviceMap sync.Map // 并发安全的 map
}

var DefaultServer = NewServer() // *Server的默认实例

// Option 客户端可选择不同的编解码器来编码消息body
type Option struct {
	MagicNum       int        // MagicNum 标记这是一个myRPC请求
	CodecType      codec.Type // choose
	ConnectTimeout time.Duration
	HandleTimeout  time.Duration
}

var DefaultOption = &Option{ // 默认选项
	MagicNum:       MagicNum,
	CodecType:      codec.GobType,
	ConnectTimeout: time.Second * 10,
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

// Register 在server中发布一组方法
func (server *Server) Register(rcvr interface{}) error {
	s := newService(rcvr)
	if _, dup := server.serviceMap.LoadOrStore(s.name, s); dup {
		return errors.New("rpc:服务已定义：" + s.name)
	}
	return nil
}

// Register 在默认Server中发布接收方的方法
func Register(rcvr interface{}) error {
	return DefaultServer.Register(rcvr)
}

// findService 通过ServiceMethod从serviceMap中找到对应的service
func (server *Server) findService(serviceMethod string) (svc *service, mtype *methodType, err error) {
	dot := strings.LastIndex(serviceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc服务器：service/method请求格式错误：" + serviceMethod)
		return
	}
	serviceName, methodName := serviceMethod[:dot], serviceMethod[dot+1:] // serviceMethod=>Service.Method分成两部分（service名+方法名）
	// 先在【serviceMap】中找到对应的service实例，再从【service实例的method】中找到对应的methodType
	svci, ok := server.serviceMap.Load(serviceName) // 读key得value
	if !ok {
		err = errors.New("rpc服务器：找不到service" + serviceName)
		return
	}
	svc = svci.(*service)
	mtype = svc.method[methodName]
	if mtype == nil {
		err = errors.New("rpc服务器：找不到method" + methodName)
	}
	return
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
		go server.handleRequest(cc, req, send, wait, time.Second*10) // 并发处理
	}
	wait.Wait() // 不wait的话直接close -> 等到break出当前for循环继而执行cc.Close，此时协程还在运行中，
	// 如果此时刚好子协程在sendResponse，但此时cc已关闭
	// 会发生runtime！整个服务器就会挂掉！
	cc.Close()
}

// request 存储call的所有信息
type request struct {
	h      *codec.Header
	argv   reflect.Value // 反射值对象（.interface()获取反射实例）
	replyv reflect.Value
	mtype  *methodType
	svc    *service
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

// readRequest 读取请求（没有请求->阻塞）
func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err // readHeader出错没救了 需要return（客户端直接挂掉
	}
	req := &request{h: h}
	req.svc, req.mtype, err = server.findService(h.ServiceMethod)
	if err != nil {
		return req, err
	}
	req.argv = req.mtype.newArgv() // 创建两个入参实例
	req.replyv = req.mtype.newReplyv()

	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface() // 确保argvi是指针->(ReadBody需要一个指针作参
	}
	/*	req.argv = reflect.New(reflect.TypeOf(""))
		if err = cc.ReadBody(req.argv.Interface()); err != nil { // 如果readBody出错了 还能返回错误给客户端（客户端还存在
			log.Println("rpc服务器：读取argv错误：", err)
		}*/
	if err = cc.ReadBody(argvi); err != nil { // cc.ReadBody将请求报文反序列化为第一个入参argv
		log.Println("rpc服务器：读取body错误：", err)
		return req, err
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

// handleRequest 处理请求=>(使用time.After()结合 select+chan 完成超时处理
func (server *Server) handleRequest(cc codec.Codec, req *request, send *sync.Mutex, wait *sync.WaitGroup, timeout time.Duration) {
	// TODO 应调用已注册的rpc方法以获得正确的replyv
	defer wait.Done()
	/*	// 简单的，目前--只需打印argv并发送hello消息
		log.Println(req.h, req.argv.Elem())                                 // reflect.Elem()通过反射获取指针指向的元素类型
		req.replyv = reflect.ValueOf(fmt.Sprintf("myrpc响应%d", req.h.Seq)) /// Seq客户端请求序列号*/
	called := make(chan struct{})
	sent := make(chan struct{}) // 为了等待sendResponse完成后再退出handle
	go func() {
		err := req.svc.call(req.mtype, req.argv, req.replyv) // call完成方法调用
		called <- struct{}{}
		if err != nil {
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, send)
			sent <- struct{}{}
			return
		}
		server.sendResponse(cc, req.h, req.replyv.Interface(), send) // 将replyv传给sendResponse完成序列化
		sent <- struct{}{}
	}()

	if timeout == 0 {
		<-called
		<-sent
		return
	}
	select {
	case <-time.After(timeout):
		req.h.Error = fmt.Sprintf("rpc服务器：请求handle超时：应在%s内", timeout)
		server.sendResponse(cc, req.h, invalidRequest, send)
	case <-called: // 读管道
		<-sent
	}
}
