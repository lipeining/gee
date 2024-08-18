package rpcx

import (
	"log"
	"net"
	"testing"
	"time"
)

type Divider struct {
}

type DividerArgs struct{ Num1, Num2 int }

func (d *Divider) Divide(args DividerArgs, reply *int) error {
	*reply = args.Num1 / args.Num2
	return nil
}

func startTestServer(addr chan string) {
	d := Divider{}
	Register(&d)

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())

	addr <- l.Addr().String()
	Accept(l)
}

func TestServer(t *testing.T) {
	addr := make(chan string)
	go startTestServer(addr)

	// in fact, following code is like a simple rpc client
	conn, _ := net.Dial("tcp", <-addr)
	defer func() { _ = conn.Close() }()

	time.Sleep(time.Second)
	cc := NewClientCodec(conn)

	// send request & receive response
	for i := 1; i <= 5; i++ {
		args := DividerArgs{i * 4, i * 2}
		_ = cc.WriteRequest(&Request{
			ServiceMethod: "Divider.Divide",
			Seq:           uint64(i),
		}, args)

		resp := new(Response)
		var reply int
		_ = cc.ReadResponseHeader(resp)
		_ = cc.ReadResponseBody(&reply)

		if reply != args.Num1/args.Num2 {
			t.Fatal("args:", args, "reply:", reply)
		}
	}
}
