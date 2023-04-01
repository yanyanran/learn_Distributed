package main

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-go/config"
	"testing"
)

type GreetingService struct {
	Greet func(ctx context.Context, name string) (string, error)
}

// Reference RPC服务ID或引用ID
func (g GreetingService) Reference() string {
	//TODO implement me
	panic("implement me")
}

func TestTokenAuth(t *testing.T) {
	config.SetConsumerService(GreetingService{})
	// 创建ReferenceConfig对象，并设置相关属性，例如接口名、直连地址等
	cfg := config.NewReferenceConfig(
		"demo.Greeter",
		config.WithProtocol("dubbo"),
		config.WithDirectUrl("127.0.0.1:20880"),
		config.WithParams(map[string]string{ // 通过WithParams方法添加Token参数,完成认证鉴权逻辑
			"token": "abc123",
		}),
	)
	gs := &GreetingService{}
	err := cfg.Assemble(gs)
	if err != nil {
		t.Error(err)
	}
	message, err := gs.Greet(context.Background(), "John")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(message)
}
