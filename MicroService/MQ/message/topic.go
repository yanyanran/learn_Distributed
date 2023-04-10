package message

import (
	"MQ/queue"
	"MQ/util"
	"context"
	"log"
)

// topic接收客户端消息，一个topic管理很多channel，然后同时发送给所有绑定的chanel上

type Topic struct {
	name                string
	newChannelChan      chan util.ChanReq
	channelMap          map[string]*Channel
	incomingMessageChan chan *Message
	msgChan             chan *Message // 有缓冲channel，消息内存队列
	readSyncChan        chan struct{}
	routerSyncChan      chan struct{}
	exitChan            chan util.ChanReq // 是否已向channel发送消息
	channelWriteStarted bool              //是否已经向channel发送消息
	backend             queue.Queue       // 添加磁盘队列【持久化处理】
}

var ( // 全局
	TopicMap     = make(map[string]*Topic)
	newTopicChan = make(chan util.ChanReq)
)

func NewTopic(name string, inMemSize int) *Topic {
	topic := &Topic{
		name:                name,
		newChannelChan:      make(chan util.ChanReq),
		channelMap:          make(map[string]*Channel),
		incomingMessageChan: make(chan *Message),
		msgChan:             make(chan *Message, inMemSize),
		readSyncChan:        make(chan struct{}),
		routerSyncChan:      make(chan struct{}),
		exitChan:            make(chan util.ChanReq),
		backend:             queue.NewDiskQueue(name),
	}
	go topic.Router(inMemSize) // 并发topic的事件处理
	return topic
}

func GetTopic(name string) *Topic {
	topicChan := make(chan interface{})
	newTopicChan <- util.ChanReq{ // 为了实现削峰 利用协程阻塞性质
		Variable: name,
		RetChan:  topicChan,
	}
	return (<-topicChan).(*Topic) // return后停止阻塞
}

func TopicFactory(ctx context.Context, inMemSize int) {
	var (
		topicReq util.ChanReq
		name     string
		topic    *Topic
		ok       bool
	)
	for { // 选择用工厂封装
		select {
		case <-ctx.Done(): // 退出
			return
		default:
			topicReq = <-newTopicChan
			name = topicReq.Variable.(string)
			if topic, ok = TopicMap[name]; !ok {
				topic = NewTopic(name, inMemSize)
				TopicMap[name] = topic
				log.Printf("TOPIC %s CREATED", name)
			}
			topicReq.RetChan <- topic // return
		}
	}
}

// GetChannel 维护channel
func (t *Topic) GetChannel(channelName string) *Channel {
	channelRet := make(chan interface{})
	t.newChannelChan <- util.ChanReq{
		Variable: channelName,
		RetChan:  channelRet,
	}
	return (<-channelRet).(*Channel)
}

func (t *Topic) PutMessage(msg *Message) {
	t.incomingMessageChan <- msg
}

func (t *Topic) MessagePump(closeChan <-chan struct{}) {
	var msg *Message
	for {
		select {
		case msg = <-t.msgChan:
		case <-t.backend.ReadReadyChan():
			bytes, err := t.backend.Get()
			if err != nil {
				log.Printf("ERROR: t.backend.Get() - %s", err.Error())
				continue
			}
			msg = NewMessage(bytes)
		case <-closeChan:
			return
		}

		t.readSyncChan <- struct{}{} // 读管道阻塞-> 加锁

		for _, channel := range t.channelMap {
			// 指定msg防止管道新消息抢占
			go func(ch *Channel, m *Message) {
				ch.PutMessage(m) // 消息推送给channel考虑原子问题(map并发读写错误)
			}(channel, msg)
		}

		t.routerSyncChan <- struct{}{} // 数据到来-> 解锁
	}
}

func (t *Topic) Close() error {
	errChan := make(chan interface{})
	t.exitChan <- util.ChanReq{
		RetChan: errChan,
	}
	err, _ := (<-errChan).(error)
	return err
}

func (t *Topic) Router(inMemSize int) {
	var (
		msg       *Message
		closeChan = make(chan struct{})
	)
	for {
		select {
		case channelReq := <-t.newChannelChan: // 添加channel topic开始推送协程给channel
			channelName := channelReq.Variable.(string)
			channel, ok := t.channelMap[channelName]
			if !ok {
				channel = NewChannel(channelName, inMemSize)
				t.channelMap[channelName] = channel
				log.Printf("TOPIC(%s): new channel(%s)", t.name, channel.name)
			}
			channelReq.RetChan <- channel
			if !t.channelWriteStarted {
				go t.MessagePump(closeChan) // topic消息推送到channel协程
				t.channelWriteStarted = true
			}

		case msg = <-t.incomingMessageChan:
			select {
			case t.msgChan <- msg:
				log.Printf("TOPIC(%s) wrote message", t.name)
			default:
				// TODO：持久化磁盘功能
				err := t.backend.Put(msg.data)
				if err != nil {
					log.Printf("ERROR: t.backend.Put() - %s", err.Error())
				}
				log.Printf("TOPIC(%s): wrote to backend", t.name)
			}
		case <-t.readSyncChan:
			<-t.routerSyncChan
		case closeReq := <-t.exitChan:
			log.Printf("TOPIC(%s): closing", t.name)

			for _, channel := range t.channelMap {
				err := channel.Close()
				if err != nil {
					log.Printf("ERROR: channel(%s) close - %s", channel.name, err.Error())
				}
			}

			t.backend.Close()
			close(closeChan)
			closeReq.RetChan <- nil
		}
	}
}
