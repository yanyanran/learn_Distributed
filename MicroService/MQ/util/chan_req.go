package util

// 定义好结构体方便后续goroutine之间通信

type ChanReq struct {
	Variable interface{}
	RetChan  chan interface{}
}

type ChanRet struct {
	Err      error
	Variable interface{}
}
