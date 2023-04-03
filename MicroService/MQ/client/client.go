package client

import "io"

type client struct {
	conn  io.ReadWriteCloser
	name  string
	state int
}
