package httpx

import "Fuse/core"

type RouterGroup struct {
	prefix string             // 公共前缀
	mws    []core.HandlerFunc // 专属中间件
	engine *Engine            // 指向底层引擎
}

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

func (group *RouterGroup) Use(mws ...core.HandlerFunc) {
	group.mws = append(group.mws, mws...)
}

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

func (group *RouterGroup) GET(pattern string, handler ...core.HandlerFunc) {
	group.addRoute(core.MethodGet, pattern, handler)
}
func (group *RouterGroup) POST(pattern string, handler ...core.HandlerFunc) {
	group.addRoute(core.MethodPost, pattern, handler)
}
func (group *RouterGroup) DELETE(pattern string, handler ...core.HandlerFunc) {
	group.addRoute(core.MethodDelete, pattern, handler)
}
func (group *RouterGroup) PUT(pattern string, handler ...core.HandlerFunc) {
	group.addRoute(core.MethodPut, pattern, handler)
}
