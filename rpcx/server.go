package rpcx

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"
)

type Server struct {
	serviceMap sync.Map
}

func NewServer() *Server {
	return &Server{}
}

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

func (server *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "CONNECT" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = io.WriteString(w, "405 must CONNECT\n")
		return
	}

	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		log.Print("rpc hijacking ", req.RemoteAddr, ": ", err.Error())
		return
	}
	_, _ = io.WriteString(conn, "HTTP/1.0 200 Connected to RPC"+"\n\n")
	server.ServeConn(conn)
}

func (server *Server) HandleHTTP(rpcPath, debugPath string) {
	http.Handle(rpcPath, server)
	http.Handle(debugPath, debugHTTP{server})
}

func (server *Server) Register(rcvr any) error {
	s := newService(rcvr)
	if _, dup := server.serviceMap.LoadOrStore(s.name, s); dup {
		return errors.New("rpc: service already defined: " + s.name)
	}
	return nil
}

func (server *Server) RegisterName(name string, rcvr any) error {
	s := newService(rcvr)
	if _, dup := server.serviceMap.LoadOrStore(name, s); dup {
		return errors.New("rpc: service already defined: " + name)
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

func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	// default use gob codec
	server.ServeCodec(NewServerCodec(conn))
}

var invalidRequest = struct{}{}

func (server *Server) ServeCodec(codec ServerCodec) {
	sending := new(sync.Mutex) // make sure to send a complete response
	wg := new(sync.WaitGroup)  // wait until all request are handled

	for {
		req, err := server.readRequest(codec)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				// already read all request
				break
			}

			log.Println("rpc: server cannot decode request: " + err.Error())
			resp := &Response{ServiceMethod: req.ServiceMethod, Seq: req.Seq, Error: err.Error()}
			server.sendResponse(codec, resp, invalidRequest, sending)
			continue
		}
		wg.Add(1)

		go server.handleRequest(codec, req, sending, wg)
	}
	wg.Wait()
	codec.Close()
}

func (server *Server) handleCallError(codec ServerCodec, req *Request, sending *sync.Mutex, err error) {
	resp := &Response{
		ServiceMethod: req.ServiceMethod,
		Seq:           req.Seq,
	}

	if err != nil {
		resp.Error = err.Error()
		server.sendResponse(codec, resp, invalidRequest, sending)
		return
	}

	server.sendResponse(codec, resp, req.replyv.Interface(), sending)
}

func (server *Server) sendResponse(codec ServerCodec, resp *Response, body any, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()

	if err := codec.WriteResponse(resp, body); err != nil {
		log.Println("rpc server: send response, got error: ", err)
	}
}

func (server *Server) readRequest(codec ServerCodec) (req *Request, err error) {
	// try read a request first
	req = new(Request)
	err = codec.ReadRequestHeader(req)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return
		}

		// cannot decode, discard body
		codec.ReadRequestBody(nil)
		return
	}

	req.service, req.mtype, err = server.findService(req.ServiceMethod)
	if err != nil {
		return
	}

	req.argv = req.mtype.newArgv()
	req.replyv = req.mtype.newReplyv()

	// make sure that argvi is a pointer,
	// ReadBody need a pointer as parameter
	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}

	err = codec.ReadRequestBody(argvi)
	return
}

func (server *Server) handleRequest(codec ServerCodec, req *Request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Println("handle request for", req.Seq, req.argv.Interface())

	if req.Timeout == 0 {
		err := req.service.call(req.mtype, req.argv, req.replyv)
		server.handleCallError(codec, req, sending, err)
		return
	}

	result := make(chan error)
	go func() {
		err := req.service.call(req.mtype, req.argv, req.replyv)
		result <- err
	}()

	select {
	case <-time.After(req.Timeout):
		err := fmt.Errorf("rpc server: request handle timeout: expect within %s", req.Timeout)
		server.handleCallError(codec, req, sending, err)
	case err := <-result:
		server.handleCallError(codec, req, sending, err)
	}
}
