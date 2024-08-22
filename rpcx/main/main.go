package main

import (
	"context"
	"log"
	"net"
	"rpcx"
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

type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func startServer(serverAddr string, registryServerAddr string) {
	regClient := registry.NewRegistry(registry.Addrs("http://127.0.0.1"+registryServerAddr+"/_rpc_/registry"), registry.Timeout(time.Minute))
	server := rpcx.NewServer(rpcx.ServerName("Foo"), rpcx.ServerRegistry(regClient))

	var foo Foo
	server.Register(&foo)

	l, _ := net.Listen("tcp", serverAddr)
	server.Accept(l)

	server.HandleHTTP("/rpc", "/debug/rpc")
}

func testTcpCall(serverAddr string) {
	conn, _ := net.Dial("tcp", serverAddr)
	defer func() { _ = conn.Close() }()

	time.Sleep(time.Second)
	cc := rpcx.NewClientCodec(conn)

	// send request & receive response
	for i := 1; i <= 5; i++ {
		args := Args{i, i}
		_ = cc.WriteRequest(&rpcx.Request{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}, args)

		resp := new(rpcx.Response)
		var reply int
		_ = cc.ReadResponseHeader(resp)
		_ = cc.ReadResponseBody(&reply)

		log.Println("call result: ", args, reply)
	}
}

func testClientCall(registryServerAddr string) {
	regClient := registry.NewRegistry(registry.Addrs("http://127.0.0.1"+registryServerAddr+"/_rpc_/registry"), registry.Timeout(time.Minute))
	client, err := rpcx.NewClient(rpcx.ClientName("Foo"), rpcx.ClientRegistry(regClient))
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second)

	// send request & receive response
	for i := 1; i <= 5; i++ {
		args := Args{i, i}
		var reply int
		err := client.Call(context.Background(), "Foo.Sum", args, &reply)
		if err != nil {
			log.Println("client call error:", err)
		}

		log.Println("client call result: ", args, reply)
	}
}

func main() {
	// startHttpRegistryClient(":8080")
	serverAddr := ":9999"
	registryServerAddr := ":8080"
	go startServer(serverAddr, registryServerAddr)
	time.Sleep(time.Second * 5)
	// testTcpCall(serverAddr)
	testClientCall(registryServerAddr)
}
