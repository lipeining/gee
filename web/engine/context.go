package engine

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type H map[string]interface{}

type Context struct {
	// origin objects
	Res http.ResponseWriter
	Req *http.Request
	// request info
	Path   string
	Method string
	// response info
	StatusCode int
	// middleware
	handlers []HandlerFunc
	index    int
}

func newContext(res http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Res:    res,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
		index:  -1,
	}
}

func (c *Context) Next() {
	c.index++
	s := len(c.handlers)
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}

func (c *Context) Param(key string) string {
	params := httprouter.ParamsFromContext(c.Req.Context())
	value := params.ByName(key)
	return value
}

func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Res.WriteHeader(code)
}

func (c *Context) SetHeader(key string, value string) {
	c.Res.Header().Set(key, value)
}

func (c *Context) String(code int, format string, values ...interface{}) {
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	c.Res.Write([]byte(fmt.Sprintf(format, values...)))
}

func (c *Context) JSON(code int, obj interface{}) {
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	encoder := json.NewEncoder(c.Res)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Res, err.Error(), 500)
	}
}

func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.Res.Write(data)
}

func (c *Context) HTML(code int, html string) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	c.Res.Write([]byte(html))
}

func (c *Context) Fail(code int, html string) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	c.Res.Write([]byte(html))
}
