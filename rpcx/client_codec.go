package rpcx

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

type ClientCodec interface {
	WriteRequest(*Request, any) error
	ReadResponseHeader(*Response) error
	ReadResponseBody(any) error

	Close() error
}

type clientCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

var _ ClientCodec = (*clientCodec)(nil)

func NewClientCodec(conn io.ReadWriteCloser) ClientCodec {
	buf := bufio.NewWriter(conn)

	return &clientCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}

func (c *clientCodec) ReadResponseHeader(header *Response) error {
	err := c.dec.Decode(header)
	if err != nil {
		log.Println("gobrpc: got error read header:", err)
	}

	return err
}

func (c *clientCodec) ReadResponseBody(body interface{}) error {
	err := c.dec.Decode(body)
	if err != nil {
		log.Println("gobrpc: got error read body:", err)
	}

	return err
}

func (c *clientCodec) WriteRequest(header *Request, body interface{}) (err error) {
	defer func() {
		// TODO: 为什么忽略错误？
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()

	if err := c.enc.Encode(header); err != nil {
		log.Println("gobrpc: got error encoding header:", err)
		return err
	}
	if err := c.enc.Encode(body); err != nil {
		log.Println("gobrpc: got error encoding body:", err)
		return err
	}

	return nil
}

func (c *clientCodec) Close() error {
	return c.conn.Close()
}
