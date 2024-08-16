package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"reflect"
	"rpc/codec"
	"strings"
	"sync"
	"time"
)

const MagicNumber = 0x3bef5c

// Server represents an RPC Server.
type Server struct {
	serviceMap sync.Map
}

const (
	connected        = "200 Connected to Gee RPC"
	defaultRPCPath   = "/_geeprc_"
	defaultDebugPath = "/debug/geerpc"
)

// NewServer returns a new Server.
func NewServer() *Server {
	return &Server{}
}

// DefaultServer is the default instance of *Server.
var DefaultServer = NewServer()

// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func Accept(lis net.Listener) { DefaultServer.Accept(lis) }

// Register publishes the receiver's methods in the DefaultServer.
func Register(rcvr interface{}) error { return DefaultServer.Register(rcvr) }

// Register publishes in the server the set of methods of the
func (server *Server) Register(rcvr interface{}) error {
	s := newService(rcvr)
	if _, dup := server.serviceMap.LoadOrStore(s.name, s); dup {
		return errors.New("rpc: service already defined: " + s.name)
	}
	return nil
}

func (server *Server) findService(serviceMethod string) (svc *service, mtype *methodType, err error) {
	names := strings.Split(serviceMethod, ".")
	if len(names) != 2 {
		err = errors.New("illegal serviceMethod format:" + serviceMethod)
		return
	}

	serviceName, methodName := names[0], names[1]
	val, ok := server.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("find service got unknown service name: " + serviceMethod)
		return
	}

	svc = val.(*service)

	mtype, ok = svc.method[methodName]
	if !ok {
		err = errors.New("find service got unknown method name: " + serviceMethod)
		return
	}

	return
}

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

	server.serveCodec(codecFactory(conn), &opt)
}

// invalidRequest
var invalidRequest = struct{}{}

func (server *Server) serveCodec(cc codec.Codec, opt *Option) {
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
		go server.handleRequest(cc, req, sending, wg, opt.HandleTimeout)
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
	req.service, req.mtype, err = server.findService(req.header.ServiceMethod)
	if err != nil {
		log.Println("rpc server: find service error:", err)
		return req, err
	}

	req.argv = req.mtype.newArgv()
	req.replyv = req.mtype.newReplyv()

	// make sure that argvi is a pointer, ReadBody need a pointer as parameter
	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}

	if err = cc.ReadBody(argvi); err != nil {
		log.Println("rpc server: read body err:", err)
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

func (server *Server) handleRequest(cc codec.Codec, req *Request, sending *sync.Mutex, wg *sync.WaitGroup, timeout time.Duration) {
	defer wg.Done()

	fmt.Println("handle request", req.header.Seq, req.argv.Interface())

	if timeout == 0 {
		err := req.service.call(req.mtype, req.argv, req.replyv)
		server.handleCallError(cc, req, sending, err)
		return
	}

	result := make(chan error)
	go func() {
		err := req.service.call(req.mtype, req.argv, req.replyv)
		result <- err
	}()

	select {
	case <-time.After(timeout):
		err := fmt.Errorf("rpc server: request handle timeout: expect within %s", timeout)
		server.handleCallError(cc, req, sending, err)
	case err := <-result:
		server.handleCallError(cc, req, sending, err)
	}
}

func (server *Server) handleCallError(cc codec.Codec, req *Request, sending *sync.Mutex, err error) {
	if err != nil {
		req.header.Error = err.Error()
		server.sendResponse(cc, req.header, invalidRequest, sending)
		return
	}

	server.sendResponse(cc, req.header, req.replyv.Interface(), sending)
}

// ServeHTTP implements an http.Handler that answers RPC requests.
func (server *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if req.Method != "CONNECT" {
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")
		res.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = io.WriteString(res, "405 must CONNECT\n")
		return
	}

	conn, _, err := res.(http.Hijacker).Hijack()
	if err != nil {
		log.Print("rpc hijacking ", req.RemoteAddr, ": ", err.Error())
		return
	}
	_, _ = io.WriteString(conn, "HTTP/1.0 "+connected+"\n\n")
	server.ServeConn(conn)
}

// HandleHTTP registers an HTTP handler for RPC messages on rpcPath.
// It is still necessary to invoke http.Serve(), typically in a go statement.
func (server *Server) HandleHTTP() {
	http.Handle(defaultRPCPath, server)
	http.Handle(defaultDebugPath, debugHTTP{server})
	log.Println("rpc server debug path:", defaultDebugPath)
}

// HandleHTTP is a convenient approach for default server to register HTTP handlers
func HandleHTTP() {
	DefaultServer.HandleHTTP()
}
