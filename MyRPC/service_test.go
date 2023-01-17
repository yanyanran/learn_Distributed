package MyRPC

import (
	"fmt"
	"reflect"
	"testing"
)

// 定义结构体Foo，实现2个方法，导出方法Sum和非导出方法sum
type Foo int

type Args struct {
	Num1, Num2 int
}

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

// 非导出
func (f Foo) sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func _assert(condition bool, msg string, v ...interface{}) {
	if !condition { // 条件
		panic(fmt.Sprintf("断言失败："+msg, v...))
	}
}

func TestNewServer(t *testing.T) {
	var foo Foo
	s := newService(&foo)
	//fmt.Println(s.name)   // Foo
	_assert(len(s.method) == 1, "错误的服务方法，应为1，但得到了%d", len(s.method))
	mType := s.method["Sum"]
	//fmt.Println(mType)  // methodType struct
	_assert(mType != nil, "方法错误，Sum不应为nil")
}

func TestMethodType_Call(t *testing.T) {
	var foo Foo
	s := newService(&foo)
	mType := s.method["Sum"]

	argv := mType.newArgv()
	replyv := mType.newReplyv()
	argv.Set(reflect.ValueOf(Args{Num1: 1, Num2: 3}))
	err := s.call(mType, argv, replyv)
	_assert(err == nil && *replyv.Interface().(*int) == 4 && mType.NumCalls() == 1, "未能调用Foo.Sum")
}
