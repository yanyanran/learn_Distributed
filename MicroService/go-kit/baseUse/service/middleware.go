package service

import (
	"context"
	"fmt"
	"go.uber.org/zap"
)

const ContextReqUUid = "req_uuid"

// NewMiddlewareServer 定义服务中间件
type NewMiddlewareServer func(Service) Service

type logMiddlewareServer struct {
	logger *zap.Logger
	next   Service
}

// NewLogMiddlewareServer 把日志记录对象嵌入中间件（其实就是对Service添加了一层装饰）
func NewLogMiddlewareServer(log *zap.Logger) NewMiddlewareServer {
	return func(service Service) Service {
		return logMiddlewareServer{
			logger: log,
			next:   service,
		}
	}
}

// TestAdd 让logMiddlewareServer实现Service中的全部方法
func (l logMiddlewareServer) TestAdd(ctx context.Context, in Add) (out AddAck) {
	defer func() {
		l.logger.Debug(fmt.Sprint(ctx.Value(ContextReqUUid)), zap.Any("调用 service logMiddlewareServer", "TestAdd"), zap.Any("req", in), zap.Any("res", out))
	}()
	out = l.next.TestAdd(ctx, in)
	return out
}
