package httpx

import (
	"Fuse/core"
	"strings"
)

type node struct {
	pattern  string  // 完整路由
	part     string  // 路由当前段
	children []*node // 子节点
	isWild   bool    // 是否模糊匹配
}

// 插入路由节点
func (n *node) insert(pattern string, parts []string, height int) {
	// 达到底部
	if len(parts) == height || n.isWild {
		n.pattern = pattern
		return
	}

	// 查找是否存在节点
	part := parts[height]
	child := n.matchChild(part)

	// 创建新节点
	if child == nil {
		child = &node{part: part, isWild: strings.HasPrefix(part, ":") || strings.HasPrefix(part, "*")}
		n.children = append(n.children, child)
	}

	// 递归插入
	child.insert(pattern, parts, height+1)

}

// 辅助函数 查看是否已经存在已创建的节点
func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part || child.isWild {
			return child
		}
	}
	return nil
}

// 查找匹配的路由节点
func (n *node) search(parts []string, height int, params map[string]string) *node {
	// 达到底部 返回
	if len(parts) == height {
		return n
	}

	part := parts[height]
	children := n.matchChildren(part)

	// 递归查找
	for _, child := range children {
		if strings.HasPrefix(child.part, "*") {
			params[child.part[1:]] = strings.Join(parts[height:], "/")
			return child
		}

		isParam := strings.HasPrefix(child.part, ":")
		if isParam {
			params[child.part[1:]] = part
		}

		result := child.search(parts, height+1, params)

		if result != nil {
			return result
		}

		// 无法找到结果 回溯删除参数字典
		if isParam {
			delete(params, child.part[1:])
		}
	}
	return nil
}

// 辅助函数 查找所有匹配的子节点
func (n *node) matchChildren(part string) []*node {
	result := make([]*node, 0)
	for _, child := range n.children {
		if child.part == part || child.isWild {
			result = append(result, child)
		}
	}
	return result
}

// 辅助函数 解析路由路径
func parsePattern(pattern string) []string {
	vs := strings.Split(pattern, "/")
	parts := make([]string, 0)
	for _, v := range vs {
		if v != "" {
			parts = append(parts, v)
			if v[0] == '*' {
				break
			}
		}
	}
	return parts
}

type Router struct {
	routes map[string]*node

	handlers map[string]core.HandlerFunc
}

func NewRouter() *Router {
	return &Router{routes: make(map[string]*node), handlers: make(map[string]core.HandlerFunc)}
}

func (r *Router) Add(method, pattern string, handler core.HandlerFunc) {
	parts := parsePattern(pattern)

	// 初始化根节点
	if _, ok := r.routes[method]; !ok {
		r.routes[method] = &node{}
	}

	r.routes[method].insert(pattern, parts, 0)

	key := method + "-" + pattern

	// 存储处理器
	r.handlers[key] = handler

}

func (r *Router) Match(method, path string) (core.HandlerFunc, map[string]string) {
	parts := parsePattern(path)
	params := make(map[string]string)
	// 查看是否有对应的方法
	root, ok := r.routes[method]
	if !ok {
		return nil, nil
	}

	// 找节点
	n := root.search(parts, 0, params)
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
