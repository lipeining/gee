package gobrpc

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
	"rpc"
)

type serverCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

func NewServerCodec(conn io.ReadWriteCloser) rpc.ServerCodec {
	buf := bufio.NewWriter(conn)

	return &serverCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}

func (c *serverCodec) ReadRequestHeader(header *rpc.Request) error {
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

func (c *serverCodec) WriteResponse(header *rpc.Response, body interface{}) (err error) {
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
