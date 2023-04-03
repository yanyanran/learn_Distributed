package test

import (
	"MQ/util"
	"fmt"
	"testing"
)

func TestUuidToStr(t *testing.T) {
	go util.UuidFactory()
	uid := <-util.UuidChan

	fmt.Printf("%c \n", uid[1])
	fmt.Println(uid)
	fmt.Printf("%s ", uid)
	fmt.Printf(uid.UuidToStr())
}
