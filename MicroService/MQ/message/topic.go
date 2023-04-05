package message

import (
	"MQ/util"
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

func TopicFactory(inMemSize int) {
	var (
		topicReq util.ChanReq
		name     string
		topic    *Topic
		ok       bool
	)
	for { // 选择用工厂封装
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

// GetChannel 维护channel
func (t *Topic) GetChannel(channelName string) *Channel {
	channelRet := make(chan interface{})
	t.newChannelChan <- util.ChanReq{
		Variable: channelName,
		RetChan:  channelRet,
	}
	return (<-channelRet).(*Channel)
}

func (t *Topic) Router(inMemSize int) {
	for {
		select {
		case channelReq := <-t.newChannelChan:
			channelName := channelReq.Variable.(string)
			channel, ok := t.channelMap[channelName]
			if !ok {
				channel = NewChannel(channelName, inMemSize)
				t.channelMap[channelName] = channel
				log.Printf("TOPIC(%s): new channel(%s)", t.name, channel.name)
			}
			channelReq.RetChan <- channel
		}
	}
}
