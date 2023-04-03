package util

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
)

// uuid生成器

type Uid []byte

var UuidChan = make(chan Uid, 1000)

func UuidFactory() { // 使用一个工厂
	for {
		UuidChan <- uuid()
	}
}

func uuid() []byte {
	b := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		log.Fatal(err)
	}
	return b
}

func (b Uid) UuidToStr() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}
