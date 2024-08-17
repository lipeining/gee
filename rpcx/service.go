package rpcx

import (
	"go/ast"
	"log"
	"reflect"
	"strings"
	"sync/atomic"
)

type methodType struct {
	method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
	numCalls  uint64
}

func (m *methodType) NumCalls() uint64 {
	return atomic.LoadUint64(&m.numCalls)
}

func (m *methodType) newArgv() reflect.Value {
	var argv reflect.Value

	// arg may be a pointer type, or a value type
	if m.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(m.ArgType.Elem())
	} else {
		argv = reflect.New(m.ArgType).Elem()
	}

	return argv
}

func (m *methodType) newReplyv() reflect.Value {
	// reply must be a pointer type
	replyv := reflect.New(m.ReplyType.Elem())

	switch m.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyType.Elem(), 0, 0))
	}

	return replyv
}

type service struct {
	name   string
	typ    reflect.Type
	rcvr   reflect.Value
	method map[string]*methodType
}

func newService(rcvr interface{}) *service {
	typ := reflect.TypeOf(rcvr)
	val := reflect.ValueOf(rcvr)
	name := reflect.Indirect(val).Type().Name()

	s := &service{
		name:   name,
		typ:    typ,
		rcvr:   val,
		method: make(map[string]*methodType),
	}

	if !ast.IsExported(s.name) {
		log.Fatalf("rpc server: %s is not a valid service name", s.name)
	}

	s.registerMethods()

	return s
}

func (s *service) registerMethods() {
	for i := 0; i < s.typ.NumMethod(); i++ {
		method := s.typ.Method(i)
		if !method.IsExported() {
			continue
		}

		// func (t *T) MethodName(argType T1, replyType *T2) error
		// index:1 is reflect to argv
		// index:2 is reflect to reply
		if method.Type.NumIn() != 3 || method.Type.NumOut() != 1 {
			// miss arg or reply, or illeagal returnType
			continue
		}

		if method.Type.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			// ensure return an error
			continue
		}

		argType, replyType := method.Type.In(1), method.Type.In(2)
		if !isExportedOrBuiltinType(argType) || !isExportedOrBuiltinType(replyType) {
			continue
		}

		m := &methodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}

		s.method[method.Name] = m
		log.Printf("rpc server: register %s.%s\n", s.name, method.Name)
	}
}

func (s *service) call(m *methodType, argv, replyv reflect.Value) error {
	atomic.AddUint64(&m.numCalls, 1)

	// get return error
	out := m.method.Func.Call([]reflect.Value{s.rcvr, argv, replyv})
	if errInter := out[0].Interface(); errInter != nil {
		return errInter.(error)
	}

	return nil
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}

func debugMethod(typ reflect.Type, method reflect.Method) {
	argv := make([]string, 0, method.Type.NumIn())
	returns := make([]string, 0, method.Type.NumOut())

	// j 从 1 开始，第 0 个入参是 reciver
	for j := 1; j < method.Type.NumIn(); j++ {
		argv = append(argv, method.Type.In(j).Name())
	}
	for j := 0; j < method.Type.NumOut(); j++ {
		returns = append(returns, method.Type.Out(j).Name())
	}

	log.Printf("func (p *%s) %s(%s) %s",
		typ.Elem().Name(),
		method.Name,
		strings.Join(argv, ","),
		strings.Join(returns, ","))
}
