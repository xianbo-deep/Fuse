package httpx

import (
	"Fuse/core"
)

type HandlerChain []core.HandlerFunc
type Router struct {
	routes map[string]*node

	handlers map[string]HandlerChain
}

func NewRouter() *Router {
	return &Router{routes: make(map[string]*node), handlers: make(map[string]HandlerChain)}
}

func (r *Router) Add(method, pattern string, handler HandlerChain) {
	// 初始化根节点
	if _, ok := r.routes[method]; !ok {
		r.routes[method] = &node{}
	}

	r.routes[method].insert(pattern, pattern)

	key := method + "-" + pattern

	// 存储处理器
	r.handlers[key] = handler

}

func (r *Router) Match(method, path string) (HandlerChain, map[string]string) {
	params := make(map[string]string)
	// 查看是否有对应的方法
	root, ok := r.routes[method]
	if !ok {
		return nil, nil
	}

	// 找节点
	n := root.search(path, params)
	if n == nil {
		return nil, nil
	}

	// 获取处理器
	key := method + "-" + n.pattern

	h, ok := r.handlers[key]
	if !ok {
		return nil, nil
	}
	return h, params
}
