package filters

import (
	"context"
	"github.com/apache/dubbo-go/protocol"
)

const TokenKey = "token"

type TokenFilter struct{}

func (tf *TokenFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	// 从invocation中获取token参数
	token := invocation.AttachmentsByKey(TokenKey, "")
	if token == "" {
		// 如果token为空，则拒绝请求
		return &protocol.RPCResult{Err: protocol.ErrDestroyedInvoker}
	}
	// 进行认证鉴权逻辑，例如验证token是否有效等
	if !authenticate(token) {
		// 如果认证失败，则拒绝请求
		return &protocol.RPCResult{Err: protocol.ErrDestroyedInvoker}
	}
	// 认证通过，继续执行后续逻辑
	return invoker.Invoke(ctx, invocation)
}

func authenticate(token string) bool {
	// TODO: 实现具体的认证鉴权逻辑，例如验证token是否有效等
	return true
}
