package message

import (
	"MQ/queue"
	"MQ/util"
	"errors"
	"io"
	"log"
	"time"
)

// consumer从channel读信息-> channel需要维护consumerMessage + 增删consumer

type Consumer interface {
	io.ReadWriter // Reader&&Writer
	GetState() int
	SetState(state int)
	String() string
	Close()
}

type Channel struct {
	name string
	// 接收增删consumer消息
	addClientChan    chan util.ChanReq
	removeClientChan chan util.ChanReq

	msgChan             chan *Message // 有缓冲 用来暂存消息
	incomingMessageChan chan *Message // 接收Provider(服务器消息
	clientMessageChan   chan *Message // 消息发送到这 然后consumer(客户端收取
	exitChan            chan util.ChanReq

	inFlightMessageChan chan *Message       // 发消息同时也往这个管道里写
	inFlightMessages    map[string]*Message // 存储已发送消息
	finishMessageChan   chan util.ChanReq   // 完成确认管道
	requeueMessageChan  chan util.ChanReq   // 重入队列

	clients []Consumer // 数组维护Client
	backend queue.Queue
}

func NewChannel(name string, inMemSize int) *Channel {
	channel := &Channel{
		name:                name,
		addClientChan:       make(chan util.ChanReq),
		removeClientChan:    make(chan util.ChanReq),
		clients:             make([]Consumer, 0, 5),
		incomingMessageChan: make(chan *Message, 5),
		msgChan:             make(chan *Message, inMemSize),
		clientMessageChan:   make(chan *Message),
		exitChan:            make(chan util.ChanReq),
		inFlightMessageChan: make(chan *Message),
		inFlightMessages:    make(map[string]*Message),
		requeueMessageChan:  make(chan util.ChanReq),
		finishMessageChan:   make(chan util.ChanReq),
		backend:             queue.NewDiskQueue(name),
	}
	go channel.Router()
	return channel
}

func (c *Channel) AddClient(client Consumer) {
	log.Printf("Channel(%s): adding client...", c.name)
	doneChan := make(chan interface{})
	c.addClientChan <- util.ChanReq{
		Variable: client,
		RetChan:  doneChan,
	}
	<-doneChan
}

func (c *Channel) RemoveClient(client Consumer) {
	log.Printf("Channel(%s): removing client...", c.name)
	doneChan := make(chan interface{})
	c.removeClientChan <- util.ChanReq{
		Variable: client,
		RetChan:  doneChan,
	}
	<-doneChan
}

func (c *Channel) PutMessage(msg *Message) {
	c.incomingMessageChan <- msg
}

func (c *Channel) PullMessage() *Message {
	return <-c.clientMessageChan
}

func (c *Channel) FinishMessage(uuidStr string) error { // 由provider确认
	errChan := make(chan interface{})
	c.finishMessageChan <- util.ChanReq{
		Variable: uuidStr,
		RetChan:  errChan,
	}
	err, _ := (<-errChan).(error)
	return err
}

func (c *Channel) pushInFlightMessage(msg *Message) {
	c.inFlightMessages[util.UuidToStr(msg.Uuid())] = msg
}

func (c *Channel) popInFlightMessage(uuidStr string) (*Message, error) {
	msg, ok := c.inFlightMessages[uuidStr]
	if !ok {
		return nil, errors.New("UUID not in flight") // ERROR
	}
	delete(c.inFlightMessages, uuidStr)
	msg.EndTimer()
	return msg, nil
}

// RequeueMessage consumer想多次消费同一条消息
func (c *Channel) RequeueMessage(uuiStr string) error {
	errChan := make(chan interface{})
	c.requeueMessageChan <- util.ChanReq{
		Variable: uuiStr,
		RetChan:  errChan,
	}
	err, _ := (<-errChan).(error)
	return err
}

func (c *Channel) Close() error {
	errChan := make(chan interface{})
	c.exitChan <- util.ChanReq{
		RetChan: errChan,
	}
	err, _ := (<-errChan).(error)
	return err
}

// Router 常驻后台goroutine-> 事件处理循环
// incomingChan -> msgChan
func (c *Channel) Router() {
	var (
		clientReq util.ChanReq
		closeChan = make(chan struct{})
	)

	go c.RequeueRouter(closeChan)
	go c.MessagePump(closeChan) // 传入closeChan防止僵尸进程出现

	for {
		select {
		case clientReq = <-c.addClientChan: // add consumer
			client := clientReq.Variable.(Consumer)
			c.clients = append(c.clients, client)
			log.Printf("CHANNEL(%s) added client success : %#v", c.name, client)
			clientReq.RetChan <- struct{}{}

		case clientReq = <-c.removeClientChan: // remove consumer
			client := clientReq.Variable.(Consumer)
			indexToRemove := -1
			for k, v := range c.clients {
				if v == client {
					indexToRemove = k
					break
				}
			}
			if indexToRemove == -1 {
				log.Printf("ERROR: could not find client(%#v) in clients(%#v)", client, c.clients)
			} else {
				c.clients = append(c.clients[:indexToRemove], c.clients[indexToRemove+1:]...)
				log.Printf("CHANNEL(%s) removed client %#v", c.name, client)
			}
			clientReq.RetChan <- struct{}{}

		case msg := <-c.incomingMessageChan:
			select {
			case c.msgChan <- msg:
				log.Printf("CHANNEL(%s) wrote message - %s", c.name, msg.Body())
			default: // 防止因 msgChan 缓冲填满时造成阻塞，加上一个 default 分支直接丢弃消息
				// TODO
				err := c.backend.Put(msg.data)
				if err != nil {
					log.Printf("ERROR: t.backend.Put() - %s", err.Error())
				}
				log.Printf("CHANNEL(%s): wrote to backend", c.name)
			}
		case closeReq := <-c.exitChan: // server退出触发
			log.Printf("CHANNEL(%s) is closing", c.name)
			close(closeChan)

			c.backend.Close()
			for _, consumer := range c.clients {
				consumer.Close()
			}

			closeReq.RetChan <- nil
		}
	}
}

// MessagePump 向 ClientMessageChan 发消息
// magChan -> ClientChan
func (c *Channel) MessagePump(closeChan chan struct{}) {
	var msg *Message
	for {
		select {
		case msg = <-c.msgChan:
		case c.backend.ReadReadyChan() <- struct{}{}:
			get, err := c.backend.Get()
			if err != nil {
				log.Printf("ERROR: t.backend.Get() - %s", err.Error())
				continue
			}
			msg := NewMessage(get)
			c.PutMessage(msg)
		case <-closeChan:
			return
		}
		if msg != nil {
			c.inFlightMessageChan <- msg
		}
		c.clientMessageChan <- msg
	}
}

func (c *Channel) RequeueRouter(closeChan chan struct{}) {
	for {
		select {
		// 消息入【已发送】队列中（超时未接收/已完成）
		case msg := <-c.inFlightMessageChan:
			// 防止provider长时间不确认消息发送完成，消息堆积在inFightMessages
			go func(msg *Message) {
				select {
				case <-time.After(30 * time.Second):
					log.Printf("CHANNEL(%s): auto requeue of message(%s)", c.name, util.UuidToStr(msg.Uuid()))
				case <-msg.timerChan: // pop后结束
					log.Printf("msg（%s）'s timerChan is exited.......", msg.Uuid())
					return
				}
				err := c.RequeueMessage(util.UuidToStr(msg.Uuid()))
				if err != nil {
					log.Printf("ERROR: channel(%s) - %s", c.name, err.Error())
				}
			}(msg)
			c.pushInFlightMessage(msg)

		// 重新入队
		case requeueReq := <-c.requeueMessageChan:
			uuidStr := requeueReq.Variable.(string)
			msg, err := c.popInFlightMessage(uuidStr)
			if err != nil {
				log.Printf("ERROR: failed to requeue message(%s) - %s", uuidStr, err.Error())
			} else {
				go func(msg *Message) {
					log.Printf("REQUEUE goruntine: %s is success requeue", uuidStr)
					c.PutMessage(msg)
				}(msg)
			}
			requeueReq.RetChan <- err

		// 消息已完成
		case finishReq := <-c.finishMessageChan:
			uuidStr := finishReq.Variable.(string)
			_, err := c.popInFlightMessage(uuidStr)
			if err != nil { // ERROR
				log.Printf("ERROR: failed to finish message(%s) - %s", uuidStr, err.Error())
			}
			finishReq.RetChan <- err
			log.Printf("FINISH: %s is success finish", uuidStr)

		case <-closeChan:
			return
		}
	}
}
