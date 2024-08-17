package rpcx

import (
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
)

type Server struct {
	serviceMap sync.Map
	reqLock    sync.Mutex // protects freeReq
	freeReq    *Request
	respLock   sync.Mutex // protects freeResp
	freeResp   *Response
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

func (server *Server) ServeCodec(codec ServerCodec) {
	// sending := new(sync.Mutex) // make sure to send a complete response
	// wg := new(sync.WaitGroup)  // wait until all request are handled

	// for {
	// 	req, err := server.readRequest(codec)
	// 	if err != nil {
	// 		if req == nil {
	// 			// it's not possible to recover, so close the connection
	// 			// already read all request
	// 			break
	// 		}
	// 		req.header.Error = err.Error()
	// 		server.sendResponse(codec, req.header, invalidRequest, sending)
	// 		continue
	// 	}
	// 	wg.Add(1)
	// 	go server.handleRequest(codec, req, sending, wg, opt.HandleTimeout)
	// }
	// wg.Wait()
	// _ = codec.Close()
}

func (server *Server) ServeRequest(codec ServerCodec) error {
	// read codec header and body
	// do
	return nil
}
