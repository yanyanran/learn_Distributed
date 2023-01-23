package singleflight

import "C"
import "sync"

// call 正在进行中或已结束的请求
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// Group 管理不同key的call
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

// Do 针对相同的key，无论Do被调用多少次，函数fn都只会被调用一次，等待fn调用结束返回值或错误
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()         // 请求进行中，等待
		return c.val, c.err // 请求结束，返回结果
	}
	c := new(call)
	c.wg.Add(1)  // 发起请求前锁+1
	g.m[key] = c // 添加到g.m，表明key已有对应的请求在处理
	g.mu.Unlock()

	c.val, c.err = fn() // 调用fn发起请求
	c.wg.Done()         // 锁-1 请求结束

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
	return c.val, c.err
}
