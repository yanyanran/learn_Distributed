package transport

import (
	"context"
	"encoding/json"
	"fmt"
	httpTransport "github.com/go-kit/kit/transport/http"
	uuid "github.com/satori/go.uuid"
	"go-kit/baseUse/end_point"
	"go-kit/baseUse/service"
	"go-kit/baseUse/utils"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

func NewHttpHandler(endpoint end_point.EndPointServer, log *zap.Logger) http.Handler {
	options := []httpTransport.ServerOption{
		httpTransport.ServerErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) {
			log.Warn(fmt.Sprint(ctx.Value(service.ContextReqUUid)), zap.Error(err))
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(errorWrapper{Error: err.Error()})
		}), // 报错走这
		//httptransport.ServerErrorHandler(NewZapLogErrorHandler(log)),

		// 为每个请求添加uuid
		httpTransport.ServerBefore(func(ctx context.Context, request *http.Request) context.Context {
			UUID := uuid.NewV5(uuid.NewV4(), "req_uuid").String()
			log.Debug("给请求添加uuid", zap.Any("UUID", UUID))
			ctx = context.WithValue(ctx, service.ContextReqUUid, UUID)
			return ctx
		}),
	}

	m := http.NewServeMux()
	// Handle-> 完成mode和server的映射
	m.Handle("/sum/", httpTransport.NewServer( // server注册
		endpoint.AddEndPoint,
		decodeHTTPADDRequest,      //解析请求值
		encodeHTTPGenericResponse, //返回值
		options...,
	))
	return m
}

func decodeHTTPADDRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var (
		in  service.Add
		err error
	)
	in.A, err = strconv.Atoi(r.FormValue("a"))
	in.B, err = strconv.Atoi(r.FormValue("b"))
	if err != nil {
		return in, err
	}
	utils.GetLogger().Debug(fmt.Sprint(ctx.Value(service.ContextReqUUid)), zap.Any(" 开始解析请求数据", in))
	return in, nil
}

func encodeHTTPGenericResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	/*	if f, ok := response.(endpoint.Failer); ok && f.Failed() != nil {
		errorEncoder(ctx, f.Failed(), w)
		return nil
	}*/
	utils.GetLogger().Debug(fmt.Sprint(ctx.Value(service.ContextReqUUid)), zap.Any("请求结束封装返回值", response))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

/*func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	fmt.Println("errorEncoder", err.Error())
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(errorWrapper{Error: err.Error()})
}*/

type errorWrapper struct {
	Error string `json:"errors"`
}
