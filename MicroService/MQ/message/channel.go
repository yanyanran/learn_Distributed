package message

import (
	"MQ/util"
	"log"
)

// consumer从channel读信息-> channel需要维护consumerMessage + 增删consumer

type Consumer interface {
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

	clients []Consumer // 数组维护Client
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

	go c.MessagePump(closeChan) // 传入closeChan防止僵尸进程出现

	for {
		select {
		case clientReq = <-c.addClientChan: // add consumer
			client := clientReq.Variable.(Consumer)
			c.clients = append(c.clients, client)
			log.Printf("CHANNEL(%s) added client %#v", c.name, client)
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
				log.Printf("CHANNEL(%s) wrote message", c.name)
			default: // 防止因 msgChan 缓冲填满时造成阻塞，加上一个 default 分支直接丢弃消息
			}
		case closeReq := <-c.exitChan:
			log.Printf("CHANNEL(%s) is closing", c.name)
			close(closeChan)

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
		case <-closeChan:
			return
		}
		c.clientMessageChan <- msg
	}
}
