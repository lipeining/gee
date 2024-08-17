package rpcx

type ServerError string

func (e ServerError) Error() string {
	return string(e)
}
