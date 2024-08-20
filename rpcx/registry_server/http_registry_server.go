package registry_server

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"
)

type httpRegistryServer struct {
	timeout time.Duration
	mu      sync.Mutex // protect following
	servers map[string]map[string]*ServerItem
}

type ServerItem struct {
	Addr  string
	start time.Time
}

const (
	defaultAddr    = ":9999"
	defaultPath    = "/_rpc_/registry"
	defaultTimeout = time.Minute * 5
)

// New create a registry instance with timeout setting
func NewHttpRegistryServer(timeout time.Duration) *httpRegistryServer {
	return &httpRegistryServer{
		servers: make(map[string]map[string]*ServerItem),
		timeout: timeout,
	}
}

var DefaultHttpRegisteryerver = NewHttpRegistryServer(defaultTimeout)

func HandleHTTP(addr string) error {
	return DefaultHttpRegisteryerver.HandleHTTP(addr, defaultPath)
}

func (r *httpRegistryServer) HandleHTTP(addr string, registryPath string) error {
	if addr == "" {
		addr = defaultAddr
	}
	l, err := net.Listen("tcp", addr)

	if err != nil {
		return err
	}

	http.Handle(registryPath, r)
	log.Println("rpc http registry server handle path:", registryPath)

	http.Serve(l, nil)
	return nil
}

func (r *httpRegistryServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		service := req.URL.Query().Get("service")
		servers := r.getServers(service)
		msg, _ := json.Marshal(servers)

		w.Header().Set("Content-Type", "application/json")
		w.Write(msg)
	case "DELETE":
		data, err := io.ReadAll(req.Body)
		if err != nil {
			log.Println("POST rpc server, read body error: ", err)
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}

		p := PostServerParam{}
		err = json.Unmarshal(data, &p)
		if err != nil {
			log.Println("POST rpc server, parse body error: ", err)
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}

		r.removeServer(p)
	case "POST":
		data, err := io.ReadAll(req.Body)
		if err != nil {
			log.Println("POST rpc server, read body error: ", err)
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}

		p := PostServerParam{}
		err = json.Unmarshal(data, &p)
		if err != nil {
			log.Println("POST rpc server, parse body error: ", err)
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}

		r.addServer(p)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

type PostServerParam struct {
	Service string `json:"service"`
	Addr    string `json:"addr"`
}

func (r *httpRegistryServer) addServer(p PostServerParam) {
	r.mu.Lock()
	defer r.mu.Unlock()

	servers := r.servers[p.Service]
	if servers == nil {
		servers = make(map[string]*ServerItem, 0)
	}

	if _, ok := servers[p.Addr]; ok {
		// update time
		servers[p.Addr].start = time.Now()
	} else {
		// add new node
		servers[p.Addr] = &ServerItem{
			Addr:  p.Addr,
			start: time.Now(),
		}
	}

	r.servers[p.Service] = servers
}

func (r *httpRegistryServer) getServers(service string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	list := make([]string, 0)
	servers := r.servers[service]
	if servers == nil {
		return list
	}

	now := time.Now()

	for addr, server := range servers {
		if r.timeout == 0 || server.start.Add(r.timeout).After(now) {
			list = append(list, addr)
		} else {
			// remove not alivable node
			delete(servers, addr)
		}
	}

	sort.Strings(list)
	return list
}

func (r *httpRegistryServer) removeServer(p PostServerParam) {
	r.mu.Lock()
	defer r.mu.Unlock()

	servers := r.servers[p.Service]
	if servers == nil {
		return
	}

	delete(servers, p.Addr)
	r.servers[p.Service] = servers
}
