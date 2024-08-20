package main

import "rpcx/registry_server"

func startHttpRegistryServer(addr string) {
	err := registry_server.HandleHTTP(addr)
	if err != nil {
		panic(err)
	}
}

func main() {
	startHttpRegistryServer(":8080")
}
