package main

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"
	"time"
)

func mainI() {
	var wg sync.WaitGroup // 反射获取结构体所有方法
	typ := reflect.TypeOf(&wg)

	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		argv := make([]string, 0, method.Type.NumIn())
		returns := make([]string, 0, method.Type.NumOut())
		// j从1开始，第0个入参是wg自己
		for j := 1; j < method.Type.NumIn(); j++ {
			argv = append(argv, method.Type.In(j).Name())
		}
		for j := 0; j < method.Type.NumOut(); j++ {
			returns = append(returns, method.Type.Out(j).Name())
		}
		log.Printf("func (w *%s) %s(%s) %s",
			typ.Elem().Name(),
			method.Name,
			strings.Join(argv, ","),
			strings.Join(returns, ","))
	}
}

/*
func (w *WaitGroup) Add(int)
func (w *WaitGroup) Done()
func (w *WaitGroup) Wait()
*/

func timeAfter() {
	// time after触发
	fmt.Println("StartTime:", time.Now())
	a := time.After(5 * time.Second)
	fmt.Println(<-a)
	fmt.Println("EndTime:", time.Now())

	// time after设置超时（3s
	ch1 := make(chan string)
	go func() {
		time.Sleep(2 * time.Second)
		ch1 <- "put value into ch1"
	}()
	select {
	case val := <-ch1:
		fmt.Println("recv value form ch1:", val)
		return
	case <-time.After(3 * time.Second):
		fmt.Println("3 seconds over, time over")
	}
}

func main() {
	// Select + Chan控制协程
	stop := make(chan bool)

	go func() {
		for {
			select {
			case <-stop: // 收到停滞信号
				fmt.Println("监控退出，停止了...")
				return
			default:
				fmt.Println("goroutine监控中...")
				time.Sleep(2 * time.Second)
			}
		}
	}()
	time.Sleep(10 * time.Second)
	fmt.Println("可以了，通知监控停止")
	stop <- true
	time.Sleep(5 * time.Second) // 检测监控是否停止

	// 子孙协程 Select + Context控制
	/*
		context.Background() 返回一个空的Context，这个空的Context一般用于整个Context树的根节点
		然后使用 context.WithCancel(parent) 创建一个可取消的子Context，当作参数传给协程使用，这样就可以使用这个子Context跟踪这个协程
	*/
	ctx, cancel := context.WithCancel(context.Background())
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done(): // 判断是否要结束
				fmt.Println("监控退出，停止了...")
				return
			default:
				fmt.Println("goroutine监控中...")
				time.Sleep(2 * time.Second)
			}
		}
	}(ctx)
	time.Sleep(10 * time.Second)
	fmt.Println("可以了，通知监控停止")
	cancel() // 发送结束指令
	time.Sleep(5 * time.Second)

	// Context 控制多个协程
	ctxII, cancelII := context.WithCancel(context.Background())
	go watch(ctxII, "【监控1】")
	go watch(ctxII, "【监控2】")
	go watch(ctxII, "【监控3】")

	time.Sleep(10 * time.Second)
	fmt.Println("可以了，通知监控停止")
	cancelII() // 3 goroutine ending
	time.Sleep(5 * time.Second)

	// WithValue传递元数据
	ctxIII, cancelIII := context.WithCancel(context.Background())
	valueCtx := context.WithValue(ctxIII, key, "【监控1】") // 附加值
	go watchValue(valueCtx)
	time.Sleep(10 * time.Second)
	fmt.Println("可以了，通知监控停止")
	cancelIII()
	time.Sleep(5 * time.Second)
}

func watch(ctx context.Context, name string) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println(name, "监控退出，停止了...")
			return
		default:
			fmt.Println(name, "goroutine监控中...")
			time.Sleep(2 * time.Second)
		}
	}
}

var key string = "name"

func watchValue(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println(ctx.Value(key), "监控退出，停止了...")
			return
		default:
			fmt.Println(ctx.Value(key), "goroutine监控中...")
			time.Sleep(2 * time.Second)
		}
	}
}
