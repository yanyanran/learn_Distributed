package end_point

import (
	"context"
	"github.com/go-kit/kit/endpoint"
	"go-kit/baseUse/service"
	"go.uber.org/zap"
)

// EndPointServer endpoint方法集合
type EndPointServer struct {
	AddEndPoint endpoint.Endpoint
}

func NewEndPointServer(s service.Service, log *zap.Logger) EndPointServer {
	var addEndPoint endpoint.Endpoint
	{
		addEndPoint = MakeAddEndPoint(s)
		// 为每个Endpoint添加日志中间件
		addEndPoint = LoggingMiddleware(log)(addEndPoint)
	}
	return EndPointServer{AddEndPoint: addEndPoint}
}

func (s EndPointServer) Add(ctx context.Context, in service.Add) service.AddAck {
	res, _ := s.AddEndPoint(ctx, in)
	return res.(service.AddAck)
}

// MakeAddEndPoint 把Service中的TestAdd转换成end_point.Endpoint
func MakeAddEndPoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(service.Add)
		res := s.TestAdd(ctx, req)
		return res, nil
	}
}
