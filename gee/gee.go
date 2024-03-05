package gee

import (
	"log"
	"net/http"
)

type HandlerFunc func(*Context)

type RouterGroup struct {
	prefix      string
	middlewares []HandlerFunc
	engine      *Engine
}

type Engine struct {
	// Engine 类型就能够使用 RouterGroup 类型的功能和属性。
	*RouterGroup
	router *router
	groups []*RouterGroup
}

func New() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}

func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engine := group.engine
	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

func (group *RouterGroup) addRoute(method string, comp string, handler HandlerFunc) {
	pattern := group.prefix + comp
	log.Printf("Route %4s - %s", method, pattern)
	group.engine.router.addRoute(method, pattern, handler)
}

func (group *RouterGroup) GET(pattern string, handler HandlerFunc) {
	group.addRoute("GET", pattern, handler)
}

func (group *RouterGroup) POST(pattern string, handler HandlerFunc) {
	group.addRoute("POST", pattern, handler)
}

func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}

// http.ListenAndServe 函数的第二个参数需要实现 http.Handler 接口。在你的代码中，engine 类型实现了 ServeHTTP 方法，因此它隐式地实现了 http.Handler 接口。

// 如果你注释掉 ServeHTTP 方法，那么 engine 类型将不再满足 http.Handler 接口的要求，因为 http.Handler 接口中定义了一个方法，即 ServeHTTP(http.ResponseWriter, *http.Request)。如果没有这个方法，编译器就无法将 engine 类型隐式地视为 http.Handler。

// 为了解决这个问题，你可以在 Run 方法中使用一个实现了 http.Handler 接口的对象，而不是直接使用 engine 对象。你可以创建一个包含 engine 的结构体，并为该结构体定义一个方法，使其满足 http.Handler 接口的要求，如下所示：

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := newContext(w, req)
	engine.router.handle(c)
}
