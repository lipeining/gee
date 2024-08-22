package rpcx

import (
	"math/rand"
	"time"
)

type SelectMode int

const (
	RandomSelect     SelectMode = iota // select randomly
	RoundRobinSelect                   // select using Robbin algorithm
)

type Balancer interface {
	Get() string
	Reset([]string)
}

func NewBalancer(mode SelectMode) Balancer {
	switch mode {
	case RandomSelect:
		return NewRandomBalancer()
	case RoundRobinSelect:
		return NewRoundRobinBalancer()
	default:
		panic("unexpected select mode")
	}
}

type randomBalancer struct {
	r       *rand.Rand
	servers []string
}

var _ Balancer = (*randomBalancer)(nil)

func NewRandomBalancer() *randomBalancer {
	return &randomBalancer{
		r: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *randomBalancer) Get() string {
	index := s.r.Intn(len(s.servers))
	return s.servers[index]
}

func (s *randomBalancer) Reset(servers []string) {
	s.servers = servers
}

type roundRobinBalancer struct {
	index   int
	servers []string
}

var _ Balancer = (*roundRobinBalancer)(nil)

func NewRoundRobinBalancer() *roundRobinBalancer {
	return &roundRobinBalancer{
		index: 0,
	}
}

func (s *roundRobinBalancer) Get() string {
	index := s.index % len(s.servers)
	server := s.servers[index]
	s.index++
	return server
}

func (s *roundRobinBalancer) Reset(servers []string) {
	s.servers = servers
}
