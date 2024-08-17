package rpcx

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

type ServerCodec interface {
	ReadRequestHeader(*Request) error
	ReadRequestBody(any) error
	WriteResponse(*Response, any) error

	// Close can be called multiple times and must be idempotent.
	Close() error
}

type serverCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

var _ ServerCodec = (*serverCodec)(nil)

func NewServerCodec(conn io.ReadWriteCloser) ServerCodec {
	buf := bufio.NewWriter(conn)

	return &serverCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}

func (c *serverCodec) ReadRequestHeader(header *Request) error {
	err := c.dec.Decode(header)
	if err != nil {
		log.Println("gobrpc: got error read header:", err)
	}

	return err
}

func (c *serverCodec) ReadRequestBody(body interface{}) error {
	err := c.dec.Decode(body)
	if err != nil {
		log.Println("gobrpc: got error read body:", err)
	}

	return err
}

func (c *serverCodec) WriteResponse(header *Response, body interface{}) (err error) {
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

func (c *serverCodec) Close() error {
	return c.conn.Close()
}
