package engine

import "net/http"

type RouterGroup struct {
	prefix      string
	middlewares []HandlerFunc
	parent      *RouterGroup
	engine      *Engine
}

func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engine := group.engine

	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		parent: group,
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)

	return newGroup
}

func (group *RouterGroup) add(method string, path string, handler HandlerFunc) {
	pattern := group.prefix + path
	group.engine.router.add(method, pattern, handler)
}

func (group *RouterGroup) Get(path string, handler HandlerFunc) {
	group.add(http.MethodGet, path, handler)
}

func (group *RouterGroup) Post(path string, handler HandlerFunc) {
	group.add(http.MethodPost, path, handler)
}

func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}
