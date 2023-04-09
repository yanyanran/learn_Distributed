package message

import "log"

type Message struct {
	data      []byte
	timerChan chan struct{}
}

func NewMessage(data []byte) *Message {
	return &Message{
		data:      data,
		timerChan: make(chan struct{}),
	}
}

func (m *Message) Uuid() []byte {
	return m.data[:16] // TODO: BUG
}

func (m *Message) Body() []byte {
	return m.data[16:]
}

func (m *Message) Data() []byte {
	return m.data
}

func (m *Message) EndTimer() {
	select {
	case m.timerChan <- struct{}{}: // 触发RequeueRouter中的msg.timerChan导致return
	default:
		log.Println("cannot write a struct into timerChan")
	}
}
