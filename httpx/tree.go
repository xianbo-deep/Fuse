package httpx

import "strings"

// 节点类型
const (
	normal uint8 = iota
	param
	wildcard
)

type node struct {
	pattern  string  // 完整路由
	path     string  // 路由当前段
	children []*node // 子节点
	nodeType uint8   // 节点类型
}

// 插入路由节点
func (n *node) insert(path string, pattern string) {
	if len(path) == 0 {
		n.pattern = pattern
		return
	}

	// 识别是否是动态节点
	var isDynamic bool
	var insertPath string
	if path[0] == '*' || path[0] == ':' {
		isDynamic = true
		// 截取到下一个'/'出现
		end := strings.IndexByte(path, '/')
		if end == -1 {
			end = len(path)
		}
		insertPath = path[:end]
	} else {
		// 识别到下一个动态文本
		isDynamic = false
		end := -1
		for i := 0; i < len(path); i++ {
			if path[i] == '*' || path[i] == ':' {
				end = i
				break
			}
		}
		// 无动态文本
		if end == -1 {
			end = len(path)
		}
		insertPath = path[:end]
	}

	// 动态部分
	if isDynamic {
		var matchChild *node
		for _, child := range n.children {
			// 是动态节点
			if child.nodeType == wildcard || child.nodeType == param {
				if child.path == insertPath {
					matchChild = child
					break
				}
				// 动态节点路径不符
				panic("route conflict")
			}
		}
		if matchChild == nil {
			nodeType := param
			if insertPath[0] == '*' {
				nodeType = wildcard
			}
			matchChild = &node{
				path:     insertPath,
				nodeType: nodeType,
			}
			n.children = append(n.children, matchChild)
		}
		matchChild.insert(path[len(insertPath):], pattern)
		return
	}
	// 计算公共前缀的长度
	i := longestCommonPrefix(insertPath, n.path)

	// 公共前缀长度小于当前节点前缀长度
	if i < len(n.path) {
		child := &node{
			path:     n.path[i:],
			children: n.children,
			nodeType: n.nodeType,
			pattern:  n.pattern,
		}

		// 更新公共前缀
		n.path = n.path[:i]
		n.children = []*node{child}
		n.pattern = ""
		n.nodeType = normal

	}

	// 当前路径还没有插入完毕
	if i < len(insertPath) {
		var matchChild *node
		searchPath := insertPath[i:]

		// 首字母
		c := searchPath[0]

		// 查找当前节点是否和剩余路径有公共前缀
		for _, child := range n.children {
			if child.nodeType == normal && child.path[0] == c {
				matchChild = child
				break
			}
		}

		// 找到了具有公共前缀的子节点
		if matchChild != nil {
			// 递归插入
			matchChild.insert(path[i:], pattern)
		} else {
			child := &node{
				path:     searchPath,
				nodeType: normal,
			}
			n.children = append(n.children, child)

			child.insert(path[len(insertPath):], pattern)
		}

	} else {
		if len(insertPath) < len(path) {
			n.insert(path[len(insertPath):], pattern)
		} else {
			// 路径匹配完毕
			n.pattern = pattern
		}
	}
}

// 返回公共前缀长度
func longestCommonPrefix(a, b string) int {
	var i int
	length := min(len(a), len(b))
	for i < length && a[i] == b[i] {
		i++
	}
	return i
}

// 查找匹配的路由节点
func (n *node) search(path string, params map[string]string) *node {
	if len(path) < len(n.path) || !strings.HasPrefix(path, n.path) {
		return nil
	}
	// 获取剩余未匹配的路径
	searchPath := path[len(n.path):]

	if len(searchPath) == 0 {
		if n.pattern == "" {
			return nil
		}
		return n
	}

	c := searchPath[0]

	// 静态查找
	var matchChild *node
	for _, child := range n.children {
		if child.nodeType == normal && child.path[0] == c {
			matchChild = child
			break
		}
	}

	if matchChild != nil {
		res := matchChild.search(searchPath, params)
		if res != nil {
			return res
		}
	}

	// 动态查找

	for _, child := range n.children {
		// 按查询参数进行查找
		if child.nodeType == param {
			end := strings.IndexByte(searchPath, '/')
			if end == -1 {
				end = len(searchPath)
			}
			// 存入参数节点
			params[child.path[1:]] = searchPath[:end]
			// 刚好走完
			if end == len(searchPath) {
				if child.pattern != "" {
					return child
				}
			} else {
				res := child.search(searchPath[end:], params)
				if res != nil {
					return res
				}
			}
			// 回溯 避免浪费内存
			delete(params, child.path[1:])
		}

		// 按通配符进行查找
		if child.nodeType == wildcard {
			params[child.path[1:]] = searchPath
			return child
		}
	}
	return nil
}
