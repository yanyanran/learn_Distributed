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

// 设计意义：利用强制类型转换，确保 struct GobCodec实现了接口Codec
// 这样IDE和编译期间就可以检查，而不是等到使用的时候
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
		g.buf.Flush()
		if err != nil {
			g.Close()
		}
	}()
	if err := g.enc.Encode(header); err != nil {
		log.Println("rpc编解码器：gob错误编码头：", err)
		return err
	}
	if err := g.enc.Encode(body); err != nil {
		log.Println("rpc编解码器：gob错误编码体：", err)
		return err
	}
	return nil
}
