package queue

// 后台队列接口，使用持久化队列保证msgChan中缓冲区满后不丢弃消息

type Queue interface {
	Get() ([]byte, error)
	Put([]byte) error
	ReadReadyChan() chan struct{}
	Close() error
}

/*func t(q Queue) {
	select {
	case q.ReadReadyChan() <- struct{}{}:
		q.Get()
	default:
	}
}*/
