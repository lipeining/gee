package rpcx

type Request struct {
	ServiceMethod string // format: "Service.Method"
	Seq           uint64 // sequence number chosen by client
	// contains filtered or unexported fields
}
