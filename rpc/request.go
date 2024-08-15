package rpc

import (
	"reflect"
	"rpc/codec"
)

type Request struct {
	header *codec.Header
	body   reflect.Value
}
