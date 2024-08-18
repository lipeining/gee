package rpcx

import (
	"reflect"
	"time"
)

type Request struct {
	ServiceMethod string        // format: "Service.Method"
	Seq           uint64        // sequence number chosen by client
	Timeout       time.Duration // timeout by client
	// contains filtered or unexported fields
	argv    reflect.Value
	replyv  reflect.Value
	mtype   *methodType
	service *service
}
