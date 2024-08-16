package rpc

type ClientCodec interface {
	WriteRequest(*Request, any) error
	ReadResponseHeader(*Response) error
	ReadResponseBody(any) error

	Close() error
}
