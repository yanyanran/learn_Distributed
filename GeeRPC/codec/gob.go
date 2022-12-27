package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

// GobCodec 结构体定义
type GobCodec struct {
	conn io.ReadWriteCloser // 构建函数传入，通过 TCP 或 Unix 建立socket时得到的链接实例
	buf  *bufio.Writer      // 为防止阻塞而创建的带缓冲的Writer (提升性能
	dec  *gob.Decoder       // gob.Encoder
	enc  *gob.Encoder       // gob.Decoder
}

var _ Codec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}

// 实现 ReadHeader、ReadBody、Write 和 Close 方法。

func (g GobCodec) Close() error {
	return g.conn.Close()
}

func (g GobCodec) ReadHeader(header *Header) error {
	return g.dec.Decode(header)
}

func (g GobCodec) ReadBody(body interface{}) error {
	return g.dec.Decode(body)
}

func (g GobCodec) Write(header *Header, body interface{}) (err error) {
	defer func() {
		_ = g.buf.Flush()
		if err != nil {
			_ = g.Close()
		}
	}()
	if err := g.enc.Encode(header); err != nil {
		log.Println("rpc codec: gob error encoding header:", err)
		return err
	}
	if err := g.enc.Encode(body); err != nil {
		log.Println("rpc codec: gob error encoding body:", err)
		return err
	}
	return nil
}
