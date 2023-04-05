# Channel设计

Channel由三个缓冲层组成：incomingChan -> msgChan -> clientChan，用户从clientChan中读取数据。

Router（处理incomingMsg -> msgChan）和它的子协程MessagePump（处理msgChan -> clientChan）和RequeueRouter负责监听管道，管道触发（往管道里面写）则由对应的方法完成。

（其中closeChan主要起到context包的作用，负责优雅地结束子协程）

如果数据长时间停留在clientChan中不被读走，则RequeueRouter中的子协程会触发Timer超时，将消息从map中delete掉，然后重新发送到incomingChan中