package rpc

import (
	"reflect"
	"rpc/codec"
)

type Request struct {
	header *codec.Header
	argv   reflect.Value
	replyv reflect.Value
	mtype  *methodType
	service    *service
}

// type Request struct {
// 	ServiceMethod string // format: "Service.Method"
// 	Seq           uint64 // sequence number chosen by client
// 	// contains filtered or unexported fields
// }
