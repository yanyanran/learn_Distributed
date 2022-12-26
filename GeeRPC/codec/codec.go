package codec

type Header struct {
	ServiceMethod string
	Seq           uint64 // 客户端选择的序列号
	Error         string
}
