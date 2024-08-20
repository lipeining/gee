package registry

import (
	"context"
	"sync"
)

type httpRegistry struct {
	opts    Options
	mu      sync.Mutex // protect following
	servers map[string]*Service
}

func NewRegistry(options ...Option) Registry {
	r := &httpRegistry{
		opts: Options{
			Context: context.Background(),
		},
	}

	err := configure(r, options...)
	if err != nil {
		return nil
	}

	return r
}

func (r *httpRegistry) String() string {
	return "HttpRegistry"
}

func (r *httpRegistry) Init(options ...Option) error {
	return configure(r, options...)
}

func configure(r *httpRegistry, options ...Option) error {
	for _, o := range options {
		o(&r.opts)
	}

	// 连接服务器 opts.Addr

	return nil
}

func (r *httpRegistry) Options() Options {
	return r.opts
}

func (r *httpRegistry) Register(*Service, ...RegisterOption) error {
	return nil
}

func (r *httpRegistry) Deregister(*Service, ...DeregisterOption) error {
	return nil
}

func (r *httpRegistry) GetService(string, ...GetOption) ([]*Service, error) {
	return nil, nil
}
