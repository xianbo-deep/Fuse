package httpx

import "github.com/xianbo-deep/Fuse/core"

// RouterGroup 路由组管理器。
//
// 维护路由公共前缀实现路由分组，并维护组内的专属中间件，同时返回路由组管理器 [RouterGroup] 支持用户挂载路由、嵌套分组、增添中间件等。
//
// 底层依赖 [Router] 路由管理器进行执行函数的高效查找。
type RouterGroup struct {
	// prefix 当前路由组的公共前缀
	prefix string
	// mws 当前路由组的专属中间件
	mws []core.HandlerFunc
	// engine 当前路由组持有的引擎
	engine *Engine
}

// Group 根据传入的前缀进行分组，返回路由组管理器 [RouterGroup]。
func (group *RouterGroup) Group(prefix string, mws ...core.HandlerFunc) *RouterGroup {
	prefix = group.prefix + prefix
	engine := group.engine
	hs := make([]core.HandlerFunc, 0, len(mws)+len(group.mws))
	hs = append(hs, group.mws...)
	hs = append(hs, mws...)
	return &RouterGroup{
		prefix: prefix,
		mws:    hs,
		engine: engine,
	}
}

// Use 对当前组使用中间件。
func (group *RouterGroup) Use(mws ...core.HandlerFunc) {
	group.mws = append(group.mws, mws...)
}

// addRoute 私有方法，用于在 [Router] 路由管理器中新增节点和函数执行链。
func (group *RouterGroup) addRoute(method string, comp string, handler HandlerChain) {
	// 完整路径
	pattern := group.prefix + comp

	// 组装完整链条
	handlers := make([]core.HandlerFunc, 0, len(handler)+len(group.mws))

	handlers = append(handlers, group.mws...)
	handlers = append(handlers, handler...)

	// 添加路由
	group.engine.router.Add(method, pattern, handlers)
}

// GET 暴露给用户挂载路由的方法。
func (group *RouterGroup) GET(pattern string, handler ...core.HandlerFunc) {
	group.addRoute(core.MethodGet, pattern, handler)
}

// POST 暴露给用户挂载路由的方法。
func (group *RouterGroup) POST(pattern string, handler ...core.HandlerFunc) {
	group.addRoute(core.MethodPost, pattern, handler)
}

// DELETE 暴露给用户挂载路由的方法。
func (group *RouterGroup) DELETE(pattern string, handler ...core.HandlerFunc) {
	group.addRoute(core.MethodDelete, pattern, handler)
}

// PUT 暴露给用户挂载路由的方法。
func (group *RouterGroup) PUT(pattern string, handler ...core.HandlerFunc) {
	group.addRoute(core.MethodPut, pattern, handler)
}
