package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

type GobCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
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

func (c *GobCodec) ReadHeader(header *Header) error {
	err := c.dec.Decode(header)
	if err != nil {
		log.Println("rpc codec: gob error read header:", err)
	}

	return err
}

func (c *GobCodec) ReadBody(body interface{}) error {
	err := c.dec.Decode(body)
	if err != nil {
		log.Println("rpc codec: gob error read body:", err)
	}

	return err
}

func (c *GobCodec) Write(header *Header, body interface{}) (err error) {
	defer func() {
		// TODO: 为什么忽略错误？
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()

	if err := c.enc.Encode(header); err != nil {
		log.Println("rpc codec: gob error encoding header:", err)
		return err
	}
	if err := c.enc.Encode(body); err != nil {
		log.Println("rpc codec: gob error encoding body:", err)
		return err
	}

	return nil
}

func (c *GobCodec) Close() error {
	return c.conn.Close()
}
