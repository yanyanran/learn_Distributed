package service

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"time"
)

type Service interface {
	TestAdd(ctx context.Context, in Add) AddAck
}

type baseServer struct {
	// 结构体实现Service接口
	logger *zap.Logger // 添加日志对象
}

func NewService(log *zap.Logger) Service {
	// 加入日志中间件
	var server Service
	server = &baseServer{log}
	server = NewLogMiddlewareServer(log)(server)
	return server
}

func (s baseServer) TestAdd(ctx context.Context, in Add) AddAck {
	//fmt.Println("A:", in.A, " B:", in.B)
	return AddAck{Res: in.A + in.B}
	//模拟耗时
	time.Sleep(time.Millisecond * 2)
	s.logger.Debug(fmt.Sprint(ctx.Value(ContextReqUUid)), zap.Any("调用 v2_service Service", "TestAdd 处理请求"))
	ack := AddAck{Res: in.A + in.B}
	s.logger.Debug(fmt.Sprint(ctx.Value(ContextReqUUid)), zap.Any("调用 v2_service Service", "TestAdd 处理请求"), zap.Any("处理返回值", ack))
	return ack
}
