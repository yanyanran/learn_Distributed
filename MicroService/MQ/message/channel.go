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
	clients          []Consumer
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

// Router 常驻后台goroutine-> 事件处理循环
func (c *Channel) Router() {
	var clientReq util.ChanReq
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
		}
	}
}
