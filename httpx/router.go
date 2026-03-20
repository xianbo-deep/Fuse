package httpx

import (
	"log"
	"strings"

	"github.com/xianbo-deep/Fuse/core"
)

// HandlerChain 中间件链，是 [core.HandlerFunc] 的切片。
type HandlerChain []core.HandlerFunc

// Router 在 HTTP 中进行路由管理。
//
// 根据不同请求方法创建不同的 radix tree，使用 map 进行维护。
//
// handlers 维护请求方法、路由匹配到的模式串组成的字符串与中间件链 [HandlerChain] 的映射，用于快速匹配中间件链。
//
// 具体来说，radix tree 做的工作是根据传入的路由从对应请求方法的 radix tree 找到匹配的模式串。
// [Router] 根据模式串和请求方法在它维护的哈希表中找到最终的函数执行链。
type Router struct {
	routes map[string]*node

	handlers map[string]HandlerChain
}

// NewRouter 初始化一个新的路由管理器。
func NewRouter() *Router {
	return &Router{routes: make(map[string]*node), handlers: make(map[string]HandlerChain)}
}

// Add 根据请求方法和路由创建节点，并存储执行链。
func (r *Router) Add(method, pattern string, handler HandlerChain) {
	// 初始化根节点
	if _, ok := r.routes[method]; !ok {
		r.routes[method] = &node{}
	}

	err := r.routes[method].insert(pattern, pattern)
	if err != nil {
		log.Printf("add %s %s error: %v", method, pattern, err)
	}
	key := method + "-" + pattern

	// 存储处理器
	r.handlers[key] = handler

}

// Match 根据传入的请求方法和真实的请求路由找到最终的函数执行链，并返回路由参数和路由参数值。
func (r *Router) Match(method, path string) (HandlerChain, map[string]string) {
	params := make(map[string]string)
	// 截断查询参数
	idx := strings.Index(path, "?")
	if idx != -1 {
		path = path[:idx]
	}
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
