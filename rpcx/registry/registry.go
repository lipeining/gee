// Package registry is an interface for service discovery
package registry

import (
	"errors"
)

var (
	// DefaultRegistry = NewRegistry()

	// Not found error when GetService is called.
	ErrNotFound = errors.New("service not found")
)

// The registry provides an interface for service discovery
// and an abstraction over varying implementations
// {consul, etcd, zookeeper, ...}.
type Registry interface {
	Init(...Option) error
	Options() Options
	Register(*Service, ...RegisterOption) error
	Deregister(*Service, ...DeregisterOption) error
	GetService(string, ...GetOption) ([]*Service, error)
	String() string
}

type Service struct {
	Name     string            `json:"name"`
	Version  string            `json:"version"`
	Metadata map[string]string `json:"metadata"`
	Nodes    []*Node           `json:"nodes"`
}

type Node struct {
	Metadata map[string]string `json:"metadata"`
	Id       string            `json:"id"`
	Address  string            `json:"address"`
}

type Option func(*Options)

type RegisterOption func(*RegisterOptions)

type DeregisterOption func(*DeregisterOptions)

type GetOption func(*GetOptions)

type ListOption func(*ListOptions)

// // Register a service node. Additionally supply options such as TTL.
// func Register(s *Service, opts ...RegisterOption) error {
// 	return DefaultRegistry.Register(s, opts...)
// }

// // Deregister a service node.
// func Deregister(s *Service) error {
// 	return DefaultRegistry.Deregister(s)
// }

// // Retrieve a service. A slice is returned since we separate Name/Version.
// func GetService(name string) ([]*Service, error) {
// 	return DefaultRegistry.GetService(name)
// }

// func String() string {
// 	return DefaultRegistry.String()
// }
