package demo

import (
	"fmt"
	"testing"
)

func TestGr(t *testing.T) {
	p := make(chan struct{}, 2)
	select {
	case p <- struct{}{}:
		fmt.Println("成功退出")
		return
	default:
	}
	<-p
}
