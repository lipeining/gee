package registry

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
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
		servers: make(map[string]*Service),
	}

	configure(r, options...)
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

	if len(r.opts.Addrs) == 0 {
		return errors.New("init registry error: addrs is empty")
	}

	// start heart beat goroutine
	go r.heartbeat()

	return nil
}

func (r *httpRegistry) heartbeat() {
	ticker := time.NewTicker(r.opts.Timeout)

	for time := range ticker.C {
		log.Println("do heartbeat at", time)
		r.doHeartbeat()
	}
}

func (r *httpRegistry) doHeartbeat() {
	r.mu.Lock()
	defer r.mu.Unlock()

	serverAddr := r.opts.Addrs[0]
	// iterate all service to update status
	for name, service := range r.servers {
		list, err := r.getServers(serverAddr, name)

		if err != nil {
			log.Println("heartbeat, got error:", name, err)
		} else {
			// update nodes
			nodes := make([]*Node, len(list))
			for _, addr := range list {
				nodes = append(nodes, &Node{
					Address: addr,
				})
			}
			service.Nodes = nodes
		}
	}
}

func (r *httpRegistry) getServers(serverAddr, name string) ([]string, error) {
	resp, err := http.Get(serverAddr + "?service=" + name)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New("get got not 200 response")
	}

	result := make([]string, 0)
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *httpRegistry) Options() Options {
	return r.opts
}

type PostServerParam struct {
	Service string `json:"service"`
	Addr    string `json:"addr"`
}

func (r *httpRegistry) Register(service *Service, options ...RegisterOption) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	opts := RegisterOptions{}
	for _, o := range options {
		o(&opts)
	}

	r.servers[service.Name] = service

	if len(service.Nodes) == 0 {
		return errors.New("register service error: nodes is empty")
	}

	serverAddr := r.opts.Addrs[0]
	address := service.Nodes[0].Address
	return r.addServer(serverAddr, service.Name, address)
}

func (r *httpRegistry) addServer(serverAddr, name, addr string) error {
	buf, err := json.Marshal(PostServerParam{
		Service: name,
		Addr:    addr,
	})
	if err != nil {
		return err
	}

	data := strings.NewReader(string(buf))
	resp, err := http.Post(serverAddr, "application/json", data)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New("add server error status:" + strconv.Itoa(resp.StatusCode))
	}

	return nil
}

func (r *httpRegistry) Deregister(service *Service, options ...DeregisterOption) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	opts := DeregisterOptions{}
	for _, o := range options {
		o(&opts)
	}

	delete(r.servers, service.Name)

	serverAddr := r.opts.Addrs[0]
	address := service.Nodes[0].Address
	return r.removeServer(serverAddr, service.Name, address)
}

func (r *httpRegistry) removeServer(serverAddr, name, addr string) error {
	buf, err := json.Marshal(PostServerParam{
		Service: name,
		Addr:    addr,
	})
	if err != nil {
		return err
	}

	data := strings.NewReader(string(buf))
	request, err := http.NewRequest("DELETE", serverAddr, data)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New("remove server error status:" + strconv.Itoa(resp.StatusCode))
	}

	return nil
}

func (r *httpRegistry) GetService(name string, options ...GetOption) ([]*Service, error) {
	var opts GetOptions
	for _, o := range options {
		o(&opts)
	}

	service := r.servers[name]
	if service != nil {
		return []*Service{service}, nil
	}

	services := make([]*Service, 0)
	serverAddr := r.opts.Addrs[0]
	list, err := r.getServers(serverAddr, name)
	if err != nil {
		return nil, err
	}

	nodes := make([]*Node, 0)
	for _, addr := range list {
		nodes = append(nodes, &Node{
			Address: addr,
		})
	}

	s := Service{
		Name:  name,
		Nodes: nodes,
	}
	services = append(services, &s)
	r.servers[name] = &s

	return services, nil
}
