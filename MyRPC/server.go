package MyRPC

import "geerpc/codec"

const MagicNum = 0x3bef5c

// Option 客户端可选择不同的编解码器来编码消息body
type Option struct {
	MagicNum  int        // MagicNum 标记这是一个myRPC请求
	CodecType codec.Type // choose
}

var DefaultOption = &Option{ // 默认选项
	MagicNum:  MagicNum,
	CodecType: codec.GobType,
}
