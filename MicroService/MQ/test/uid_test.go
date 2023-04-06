package test

import (
	"MQ/util"
	"fmt"
	"testing"
)

func TestUuidToStr(t *testing.T) {
	go util.UuidFactory()
	uid := <-util.UuidChan

	for i := 0; i < 16; i++ {
		fmt.Printf("%c ", uid[i])
	}
	//println(uid[1])
	//	println(uid)

	fmt.Println(uid)
	fmt.Printf("%s ", uid)
	//fmt.Printf(uid.UuidToStr())
	fmt.Println("11111111111111111111")

	a := "ABCD"
	S1 := []byte(a)
	fmt.Println(S1)
}
