package protocol

import (
	"MQ/message"
	"MQ/util"
	"bufio"
	"bytes"
	"context"
	"log"
	"reflect"
	"strings"
)

/*
 SUB（订阅）、GET（读取）、FIN（完成）和 REQ （重入）
*/

type Protocol struct {
	channel *message.Channel
}

func (p *Protocol) IOLoop(ctx context.Context, client StatefulReadWriter) error {
	var (
		err  error
		line string
		resp []byte
	)
	client.SetState(ClientInit)
	reader := bufio.NewReader(client)
	for { // 循环从client逐行获取
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line, err = reader.ReadString('\n')
		log.Printf("[IOLOOP fine client] %s SUCEESS", client.GetName())
		if err != nil {
			//p.channel.RemoveClient(client)
			break
		}

		// replace 返回将s中前n个不重叠old子串都替换为new的新字符串，如果n<0会替换所有old子串
		line = strings.Replace(line, "\n", " ", -1)
		//line = strings.Replace(line, "\r", " ", -1)
		params := strings.Split(line, " ") // 拆分成各个参数传给Execute

		log.Printf("PROTOCOL: %#v", params)

		resp, err = p.Execute(client, params...)
		if err != nil {
			/*			_, err = client.Write([]byte(err.Error()))
						if err != nil {
							break
						}*/
			log.Println("[ERROR] IOLoop ERROR: ", err.Error())
			continue
		}

		if resp != nil {
			_, err = client.Write(resp)
			if err != nil {
				p.channel.RemoveClient(client)
				break
			}
		}
		log.Printf("[PROTOCAL]: client %s exiting IOLoop", client)
		p.channel.RemoveClient(client)
		client.Close()
	}
	return err
}

// Execute 使用反射（/switch）为该命令调用适当的方法
func (p *Protocol) Execute(client StatefulReadWriter, params ...string) ([]byte, error) {
	var (
		err  error
		resp []byte
	)

	// 以传入的 params 的第一项作为方法名，判断有无实现该函数并执行发射调用
	typ := reflect.TypeOf(p)
	args := make([]reflect.Value, 3)
	args[0] = reflect.ValueOf(p)
	args[1] = reflect.ValueOf(client)

	cmd := strings.ToUpper(params[0])

	if method, ok := typ.MethodByName(cmd); ok {
		log.Printf("[Execute]: " + cmd)
		args[2] = reflect.ValueOf(params)
		returnValues := method.Func.Call(args) // 传value启动Call

		if !returnValues[0].IsNil() {
			resp = returnValues[0].Interface().([]byte)
		}

		if !returnValues[1].IsNil() {
			err = returnValues[1].Interface().(error)
		}
		return resp, err
	}
	return nil, ClientErrInvalid
}

// SUB 获取topic再获取channel 最后绑定客户端连接和channel
func (p *Protocol) SUB(client StatefulReadWriter, params []string) ([]byte, error) {
	if client.GetState() != ClientInit {
		return nil, ClientErrInvalid
	}

	if len(params) < 3 {
		return nil, ClientErrInvalid
	}

	topicName := params[1]
	if len(topicName) == 0 {
		return nil, ClientErrBadTopic
	}

	channelName := params[2]
	if len(channelName) == 0 {
		return nil, ClientErrBadChannel
	}

	client.SetState(ClientWaitGet)

	topic := message.GetTopic(topicName)
	p.channel = topic.GetChannel(channelName)
	p.channel.AddClient(client)
	return nil, nil
}

// GET 向绑定的 channel 发送消息，然后修改状态
func (p *Protocol) GET(client StatefulReadWriter, params []string) ([]byte, error) {
	if client.GetState() != ClientWaitGet {
		return nil, ClientErrInvalid
	}

	msg := p.channel.PullMessage()
	if msg == nil {
		log.Printf("ERROR: msg == nil")
		return nil, ClientErrBadMessage
	}

	uuidStr := util.UuidToStr(msg.Uuid())
	log.Printf("PROTOCOL: writing msg(%s) to client(%s) - %s", uuidStr, client.String(), string(msg.Body()))

	client.SetState(ClientWaitResponse)

	return msg.Data(), nil
}

func (p *Protocol) FIN(client StatefulReadWriter, params []string) ([]byte, error) {
	if client.GetState() != ClientWaitResponse {
		return nil, ClientErrInvalid
	}

	if len(params) < 2 {
		return nil, ClientErrInvalid
	}

	uuidStr := params[1]
	err := p.channel.FinishMessage(uuidStr)
	if err != nil {
		client.SetState(ClientWaitGet)
		return nil, err
	}

	client.SetState(ClientWaitGet)

	return nil, nil
}

func (p *Protocol) REQ(client StatefulReadWriter, params []string) ([]byte, error) {
	if client.GetState() != ClientWaitResponse {
		return nil, ClientErrInvalid
	}

	if len(params) < 2 {
		return nil, ClientErrInvalid
	}

	uuidStr := params[1]
	err := p.channel.RequeueMessage(uuidStr)
	if err != nil {
		return nil, err
	}

	client.SetState(ClientWaitGet)

	return nil, nil

}

// PUB 服务端写入消息，与topic交互
func (p *Protocol) PUB(client StatefulReadWriter, params []string) ([]byte, error) {
	var buf bytes.Buffer
	var err error

	if client.GetState() != -1 { // 假客户端无法访问ClientInit 初始化为-1
		return nil, ClientErrInvalid
	}
	if len(params) < 3 {
		return nil, ClientErrInvalid
	}

	topicName := params[1]
	body := []byte(params[2])
	_, err = buf.Write(<-util.UuidChan)
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(body)
	if err != nil {
		return nil, err
	}

	topic := message.GetTopic(topicName)
	topic.PutMessage(message.NewMessage(buf.Bytes()))
	return []byte("OK"), nil
}
