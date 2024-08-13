package engine

import (
	"net/http"
)

type Engine struct {
	router *Router
	*RouterGroup
	groups []*RouterGroup
}

func New() *Engine {
	router := newRouter()
	engine := &Engine{router: router}
	router.setEngine(engine)

	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}

	return engine
}

func (engine *Engine) Get(path string, handler HandlerFunc) {
	engine.router.Get(path, handler)
}

func (engine *Engine) Post(path string, handler HandlerFunc) {
	engine.router.Post(path, handler)
}

func (engine *Engine) Put(path string, handler HandlerFunc) {
	engine.router.Put(path, handler)
}

func (engine *Engine) Delete(path string, handler HandlerFunc) {
	engine.router.Delete(path, handler)
}

func (engine *Engine) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	engine.router.ServeHTTP(res, req)
}

func (engine *Engine) Run(address string) {
	http.ListenAndServe(address, engine)
}
