package rpcx

import "log"

type Call struct {
	ServiceMethod string // The name of the service and method to call.
	Seq           uint64
	Args          any        // The argument to the function (*struct).
	Reply         any        // The reply from the function (*struct).
	Error         error      // After completion, the error status.
	Done          chan *Call // Receives *Call when Go is complete.
}

func (call *Call) done() {
	if call.Error != nil {
		log.Println("call done, has error:", call.Error)
	}
	call.Done <- call
}
