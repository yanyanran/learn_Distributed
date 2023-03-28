package service

import "context"

type Service interface {
	TestAdd(ctx context.Context, in Add) AddAck
}

type baseServer struct {
	// 结构体实现Service接口
}

func NewService() Service {
	return &baseServer{}
}

func (s baseServer) TestAdd(ctx context.Context, in Add) AddAck {
	return AddAck{Res: in.A + in.B}
}
