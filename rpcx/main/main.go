package main

import (
	"log"
	"rpcx/registry"
	"time"
)

func startHttpRegistryClient(addr string) {
	regClient := registry.NewRegistry(registry.Addrs("http://127.0.0.1"+addr+"/_rpc_/registry"), registry.Timeout(time.Minute))
	nodes := make([]*registry.Node, 0)
	nodes = append(nodes, &registry.Node{
		Address: "tcp::9999",
	})

	service := "hello"
	err := regClient.Register(&registry.Service{
		Name:  service,
		Nodes: nodes,
	})
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 5)

	list, err := regClient.GetService(service)
	if err != nil {
		log.Println("GetService error:", err)
	}
	for _, s := range list {
		log.Println("GetService service:", *s)
	}
}

func main() {
	startHttpRegistryClient(":8080")
}
