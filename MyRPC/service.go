package MyRPC

import (
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

/*
通过反射实现结构体与服务的映射关系
*/
type methodType struct {
	method    reflect.Method // 方法本身
	ArgType   reflect.Type   // 第一个参数类型
	ReplyType reflect.Type   // 第二个参数类型
	numCalls  uint64         // 后续统计方法调用次数时会用到
}

func (m *methodType) NumCalls() uint64 {
	return atomic.LoadUint64(&m.numCalls) // atomic原子操作
}

func (m *methodType) newArgv() reflect.Value {
	var argv reflect.Value
	if m.ArgType.Kind() == reflect.Ptr { // arg--指针类型/值类型
		argv = reflect.New(m.ArgType.Elem())
	} else {
		argv = reflect.New(m.ArgType).Elem()
	}
	return argv
}

func (m *methodType) newReplyv() reflect.Value {
	replyv := reflect.New(m.ReplyType.Elem()) // reply必须是指针类型
	switch m.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyType.Elem(), 0, 0))
	}
	return replyv
}

type service struct {
	name   string                 // 映射的结构体的名称
	typ    reflect.Type           // 结构体类型
	rcvr   reflect.Value          // 调用时需要rcvr作为第0个参数
	method map[string]*methodType // map存储映射的结构体的所有符合条件的方法
}

func newService(rcvr interface{}) *service { // 入参是任意需要映射为服务的struct实例
	s := new(service) // new个新服务
	s.rcvr = reflect.ValueOf(rcvr)
	s.name = reflect.Indirect(s.rcvr).Type().Name() // Indirect(v)用于获取v指向的值
	//s.typ = s.rcvr.Type()
	s.typ = reflect.TypeOf(rcvr)
	if !ast.IsExported(s.name) {
		log.Fatalf("rpc服务器：%s不是有效的服务名称", s.name)
	}
	s.registerMethods()
	return s
}

// registerMethods 过滤出符合条件的方法：
// 2个导出或内置类型的入参（反射时为3个，第0个是自身，类似于java中的this）
// 返回值有且只有1个，类型为error
func (s *service) registerMethods() {
	s.method = make(map[string]*methodType)
	for i := 0; i < s.typ.NumMethod(); i++ {
		method := s.typ.Method(i)
		mType := method.Type
		/*
			reflect.Type.NumIn()：获取函数参数个数
			reflect.Type.In(i)：获取第i个参数的reflect.Type
			reflect.Type.NumOut()：获取函数返回值个数
			reflect.Type.Out(i)：获取第i个返回值的reflect.Type
		*/
		if mType.NumIn() != 3 || mType.NumOut() != 1 {
			continue
		}
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}
		argType := mType.In(1)
		replyType := mType.In(2)
		if !isExportedOrBuiltinType(argType) || !isExportedOrBuiltinType(replyType) {
			continue
		}
		s.method[method.Name] = &methodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}
		log.Printf("rpc服务器：注册%s.%s", s.name, method.Name)
	}
}

func isExportedOrBuiltinType(typ reflect.Type) bool {
	return ast.IsExported(typ.Name()) || typ.PkgPath() == ""
}

// 通过反射值调用方法
func (s *service) call(m *methodType, argv, replyv reflect.Value) error {
	atomic.AddUint64(&m.numCalls, 1)
	f := m.method.Func
	returnValues := f.Call([]reflect.Value{s.rcvr, argv, replyv})
	if err := returnValues[0].Interface(); err != nil {
		return err.(error)
	}
	return nil
}
