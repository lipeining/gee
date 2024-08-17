package rpcx

import (
	"io"
	"net"
)

const (
	defaultRPCPath   = "/_rpc_"
	defaultDebugPath = "/debug/rpc"
)

// DefaultServer is the default instance of *Server.
var DefaultServer = NewServer()

func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}

func HandleHTTP() {
	DefaultServer.HandleHTTP(defaultRPCPath, defaultDebugPath)
}

func Register(rcvr interface{}) error {
	return DefaultServer.Register(rcvr)
}

func RegisterName(name string, rcvr any) error {
	return DefaultServer.RegisterName(name, rcvr)
}

func ServeCodec(codec ServerCodec) {
	DefaultServer.ServeCodec(codec)
}

func ServeConn(conn io.ReadWriteCloser) {
	DefaultServer.ServeConn(conn)
}

func ServeRequest(codec ServerCodec) error {
	return DefaultServer.ServeRequest(codec)
}
