package server

import (
	"MQ/protocol"
	"context"
	"encoding/binary"
	"io"
	"log"
)

// 客户端处理函数，处理client和server之间的TCP连接

type Client struct {
	conn  io.ReadWriteCloser
	name  string
	state int // 连接状态
}

func NewClient(conn io.ReadWriteCloser, name string) *Client {
	return &Client{conn, name, -1}
}

func (c *Client) String() string {
	return c.name
}

func (c *Client) GetName() string {
	return c.name
}

func (c *Client) GetState() int {
	return c.state
}

func (c *Client) SetState(state int) {
	c.state = state
}

func (c *Client) Read(data []byte) (int, error) {
	return c.conn.Read(data)
}

// 给client写消息之前先往连接中写入消息体长度（固定4字节）-> client读取就能先读取长度再按长度读信息
func (c *Client) Write(data []byte) (int, error) {
	var err error
	err = binary.Write(c.conn, binary.BigEndian, int32(len(data))) // binary.BigEndian
	if err != nil {
		return 0, err
	}

	n, err := c.conn.Write(data)
	if err != nil {
		return 0, err
	}
	return n + 4, nil
}

func (c *Client) Close() {
	log.Printf("CLIENT(%s): closing", c.String())
	c.conn.Close()
}

// Handle 从客户端读取数据，保持状态并做出响应
func (c *Client) Handle(ctx context.Context) {
	defer c.Close()
	proto := &protocol.Protocol{}
	err := proto.IOLoop(ctx, c)
	if err != nil {
		log.Printf("ERROR: client(%s) - %s", c.String(), err.Error())
		return
	}
}
