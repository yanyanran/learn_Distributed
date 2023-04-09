package main

import (
	"MQ/client"
	"MQ/util"
	"log"
)

// 消费者测试代码

func main() {
	consumeClient := client.NewClient(nil)
	err := consumeClient.Connect("127.0.0.1", 5151)
	if err != nil {
		log.Fatal(err)
	}
	consumeClient.WriteCommand(consumeClient.Subscribe("test", "ch")) // SUB
	for {
		msg, err := consumeClient.ReadResponse()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%s - %s", util.UuidToStr(msg.Uuid()), msg.Body()) // TODO: BUG
		err = consumeClient.WriteCommand(consumeClient.Finish(util.UuidToStr(msg.Uuid())))
		if err != nil {
			log.Println("[ERROR]consumer write command error!!!!")
			log.Fatal(err)
		}
	}
}
