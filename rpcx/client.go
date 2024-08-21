package rpcx

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"sync"
)

type Client struct {
	codec    ClientCodec
	mu       sync.Mutex
	seq      uint64
	pending  map[uint64]*Call
	closing  bool // user has called Close
	shutdown bool // server has told us to stop
}

var _ io.Closer = (*Client)(nil)
var ErrShutdown = errors.New("connection is shut down")

func Dial(network, address string) (*Client, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}

	return NewClient(conn), nil
}

// func DialHTTP(network, address string) (*Client, error) {
// 	return nil, nil
// }

func NewClient(conn io.ReadWriteCloser) *Client {
	return NewClientWithCodec(NewClientCodec(conn))
}

func NewClientWithCodec(codec ClientCodec) *Client {
	client := &Client{
		codec:   codec,
		pending: make(map[uint64]*Call),
		seq:     1,
	}

	go client.receive()

	return client
}

func (client *Client) Call(ctx context.Context, serviceMethod string, args any, reply any) error {
	call := client.Go(serviceMethod, args, reply, make(chan *Call, 1))

	select {
	case <-ctx.Done():
		client.removeCall(call.Seq)
		return errors.New("rpc client: call failed: " + ctx.Err().Error())
	case call := <-call.Done:
		return call.Error
	}
}

func (client *Client) Go(serviceMethod string, args any, reply any, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 1)
	} else if cap(done) == 0 {
		log.Panic("rpc client: done channel is unbuffered")
	}

	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}

	client.send(call)
	return call
}

func (client *Client) send(call *Call) error {
	seq, err := client.registerCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return nil
	}

	req := &Request{
		ServiceMethod: call.ServiceMethod,
		Seq:           seq,
	}

	log.Println("rpc client: send req:", seq, "args:", call.Args)
	err = client.codec.WriteRequest(req, call.Args)
	if err != nil {
		call := client.removeCall(seq)
		// call may be nil, it usually means that Write partially failed,
		// client has received the response and handled
		if call != nil {
			call.Error = err
			call.done()
		}
		return err
	}

	return nil
}

func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.closing {
		return ErrShutdown
	}
	client.closing = true

	return client.codec.Close()
}

func (client *Client) receive() {
	var err error

	for err == nil {
		resp := new(Response)
		if err = client.codec.ReadResponseHeader(resp); err != nil {
			log.Println("client receive got response header error:", err)
			break
		}

		call := client.removeCall(resp.Seq)
		if call == nil {
			// call is missing
			err = client.codec.ReadResponseBody(nil)
			break
		}

		if resp.Error != "" {
			// call got error
			call.Error = errors.New(resp.Error)
			err = client.codec.ReadResponseBody(nil)
			call.done()
			break
		}

		// read reply(pointer), set call reply
		err = client.codec.ReadResponseBody(call.Reply)
		if err != nil {
			call.Error = errors.New("read body error " + err.Error())
		}
		call.done()
	}

	client.terminateCalls(err)
}

// IsAvailable return true if the client does work
func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()

	return !client.shutdown && !client.closing
}

func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	call.Seq = client.seq
	// add to pending map
	client.pending[call.Seq] = call
	client.seq++

	return call.Seq, nil
}

func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()

	call, ok := client.pending[seq]
	if ok {
		delete(client.pending, seq)
	}

	return call
}

func (client *Client) terminateCalls(err error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	client.shutdown = true
	// set call error
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}
