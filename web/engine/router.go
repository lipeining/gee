package engine

import (
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

type HandlerFunc func(*Context)

type Router struct {
	*httprouter.Router
	engine *Engine
}

func newRouter() *Router {
	r := httprouter.New()
	return &Router{r, nil}
}

func (router *Router) setEngine(e *Engine) {
	router.engine = e
}

func (router *Router) add(method string, path string, handler HandlerFunc) {
	router.HandlerFunc(method, path, func(res http.ResponseWriter, req *http.Request) {
		var middlewares []HandlerFunc
		for _, group := range router.engine.groups {
			if strings.HasPrefix(req.URL.Path, group.prefix) {
				middlewares = append(middlewares, group.middlewares...)
			}
		}

		c := newContext(res, req)
		c.handlers = append(middlewares, handler)

		c.Next()
	})
}

func (router *Router) Get(path string, handler HandlerFunc) {
	router.add(http.MethodGet, path, handler)
}

func (router *Router) Post(path string, handler HandlerFunc) {
	router.add(http.MethodPost, path, handler)
}

func (router *Router) Put(path string, handler HandlerFunc) {
	router.add(http.MethodPut, path, handler)
}

func (router *Router) Delete(path string, handler HandlerFunc) {
	router.add(http.MethodDelete, path, handler)
}
