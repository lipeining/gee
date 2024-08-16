package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"rpc/codec"
	"sync"
	"time"
)

// Call represents an active RPC.
type Call struct {
	Seq           uint64
	ServiceMethod string     // The name of the service and method to call.
	Args          any        // The argument to the function (*struct).
	Reply         any        // The reply from the function (*struct).
	Error         error      // After completion, the error status.
	Done          chan *Call // Receives *Call when Go is complete.
}

func (call *Call) done() {
	if call.Error != nil {
		log.Println("call done, has error:", call.Error)
	}
	call.Done <- call
}

// Client represents an RPC Client.
// There may be multiple outstanding Calls associated
// with a single Client, and a Client may be used by
// multiple goroutines simultaneously.
type Client struct {
	cc       codec.Codec
	opt      *Option
	mu       sync.Mutex // protect following
	seq      uint64
	pending  map[uint64]*Call
	closing  bool // user has called Close
	shutdown bool // server has told us to stop
}

var _ io.Closer = (*Client)(nil)

var ErrShutdown = errors.New("connection is shut down")

type newClientFunc func(conn net.Conn, opt *Option) (client *Client, err error)

type clientResult struct {
	client *Client
	err    error
}

func NewClient(conn net.Conn, opt *Option) (*Client, error) {
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		err := fmt.Errorf("invalid codec type %s", opt.CodecType)
		log.Println("rpc client: codec error:", err)
		return nil, err
	}

	// send json header
	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("rpc client: options error: ", err)
		conn.Close()
		return nil, err
	}

	client := &Client{
		cc:      codec.NewGobCodec(conn),
		opt:     opt,
		pending: make(map[uint64]*Call),
		seq:     1,
	}
	go client.receive()

	return client, nil
}

func (client *Client) receive() {
	var err error

	for err == nil {
		var h codec.Header
		if err = client.cc.ReadHeader(&h); err != nil {
			log.Println("client receive got error:", err)
			break
		}

		call := client.removeCall(h.Seq)
		if call == nil {
			// call is missing
			err = client.cc.ReadBody(nil)
			break
		}

		if h.Error != "" {
			// call got error
			call.Error = errors.New(h.Error)
			err = client.cc.ReadBody(nil)
			call.done()
			break
		}

		// read reply(pointer), set call reply
		err = client.cc.ReadBody(call.Reply)
		if err != nil {
			call.Error = errors.New("read body error " + err.Error())
		}
		call.done()
	}

	client.terminateCalls(err)
}

func dialTimeout(f newClientFunc, network, address string, opts ...*Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTimeout(network, address, opt.ConnectTimeout)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()

	ch := make(chan clientResult)
	go func() {
		client, err := f(conn, opt)
		ch <- clientResult{client: client, err: err}
	}()

	if opt.ConnectTimeout == 0 {
		result := <-ch
		return result.client, result.err
	}

	select {
	case <-time.After(opt.ConnectTimeout):
		return nil, fmt.Errorf("rpc client: connect timeout: expect within %s", opt.ConnectTimeout)
	case result := <-ch:
		return result.client, result.err
	}
}

func Dial(network, address string, opts ...*Option) (*Client, error) {
	return dialTimeout(NewClient, network, address, opts...)
}

func (client *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
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

// Call invokes the named function, waits for it to complete,
// and returns its error status.
func (client *Client) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	call := client.Go(serviceMethod, args, reply, make(chan *Call, 1))

	select {
	case <-ctx.Done():
		client.removeCall(call.Seq)
		return errors.New("rpc client: call failed: " + ctx.Err().Error())
	case call := <-call.Done:
		return call.Error
	}
}

func (client *Client) send(call *Call) {
	// register this call.
	seq, err := client.registerCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return
	}

	// prepare request header
	h := codec.Header{
		ServiceMethod: call.ServiceMethod,
		Seq:           seq,
		Error:         "",
	}

	log.Println("rpc client: send req:", seq, "args:", call.Args)
	if err := client.cc.Write(&h, call.Args); err != nil {
		call := client.removeCall(seq)
		// call may be nil, it usually means that Write partially failed,
		// client has received the response and handled
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

// Close the connection
func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.closing {
		return ErrShutdown
	}
	client.closing = true

	return client.cc.Close()
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
