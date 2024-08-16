package rpc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"rpc/codec"
	"sync"
)

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int        // MagicNumber marks this's a gee rpc request
	CodecType   codec.Type // client may choose different Codec to encode body
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}



// Server represents an RPC Server.
type Server struct{}

// NewServer returns a new Server.
func NewServer() *Server {
	return &Server{}
}

// DefaultServer is the default instance of *Server.
var DefaultServer = NewServer()

// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func Accept(lis net.Listener) { DefaultServer.Accept(lis) }

// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		go server.ServeConn(conn)
	}
}

func (server *Server) ServeConn(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}

	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
		return
	}

	codecFactory, ok := codec.NewCodecFuncMap[opt.CodecType]
	if !ok {
		log.Println("rpc server: options.CodeType is missing", opt)
		return
	}

	server.serveCodec(codecFactory(conn))
}

// invalidRequest
var invalidRequest = struct{}{}

func (server *Server) serveCodec(cc codec.Codec) {
	sending := new(sync.Mutex) // make sure to send a complete response
	wg := new(sync.WaitGroup)  // wait until all request are handled

	for {
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				// it's not possible to recover, so close the connection
				// already read all request
				break
			}
			req.header.Error = err.Error()
			server.sendResponse(cc, req.header, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		go server.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
	_ = cc.Close()
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

func (server *Server) readRequest(cc codec.Codec) (*Request, error) {
	header, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}

	req := &Request{header: header}

	req.body = reflect.New(reflect.TypeOf(""))
	if err := cc.ReadBody(req.body.Interface()); err != nil {
		log.Println("rpc server: read request body error:", err)
		return req, err
	}

	return req, nil
}

func (server *Server) sendResponse(cc codec.Codec, header *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()

	if err := cc.Write(header, body); err != nil {
		log.Println("rpc server: send response, got error: ", err)
	}
}

func (server *Server) handleRequest(cc codec.Codec, req *Request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Println("handle request", req.header.Seq, req.body.Elem().String())
	resp := reflect.ValueOf(fmt.Sprintf("rpc resp %d", req.header.Seq))
	server.sendResponse(cc, req.header, resp.Interface(), sending)
}
