package httpx

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/xianbo-deep/Fuse/core"
)

// Engine 是 Http 模块的底层引擎。
//
// 它负责业务逻辑的处理和中间件调用。
type Engine struct {
	router *Router
	pool   sync.Pool
	*RouterGroup
	trustedProxies []*net.IPNet
}

func New() *Engine {
	e := &Engine{
		router: NewRouter(),
	}
	e.RouterGroup = &RouterGroup{
		prefix: "",
		mws:    make([]core.HandlerFunc, 0),
		engine: e,
	}
	e.pool.New = func() any {
		c := NewCtx(context.Background(), e)
		return c
	}
	return e
}

// ServeHTTP HTTP 协议下框架执行的核心逻辑。
//
// 引擎 [Engine] 实现这个方法才可以传入 [http.Server] 执行
func (e *Engine) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// 创建上下文
	c := e.pool.Get().(*Ctx)

	// 传入原生上下文
	c.WithContext(request.Context())

	defer func() {
		// 清空上下文状态
		c.reset()

		// 回收上下文
		e.pool.Put(c)
	}()

	// 设置协议
	if strings.ToLower(request.Header.Get("Upgrade")) == "websocket" {
		c.Set(core.CtxKeyProtocol, core.ProtocolWS)
	} else if strings.Contains(request.Header.Get("Accept"), "text/event-stream") {
		c.Set(core.CtxKeyProtocol, core.ProtocolSSE)
	} else {
		c.Set(core.CtxKeyProtocol, core.ProtocolHTTP)
	}

	// 换成包装后的writer
	rw := core.NewResponseWriter(writer)

	// 读取输入
	c.Writer = rw
	c.Request = request
	c.Set(core.CtxKeyMethod, request.Method)
	c.Set(core.CtxKeyPath, request.URL.Path)

	// 根据请求路径匹配业务方法
	hs, params := e.router.Match(request.Method, request.URL.Path)
	if hs == nil {
		c.Err(errors.New("can not find handler with current route"))
		c.Render(core.Fail(core.CodeNotFound, "未找到路由"))
	}

	// 记录路径参数映射表
	for k, v := range params {
		c.Set("param-"+k, v)
	}
	// 组装中间件
	c.handlers = append(c.handlers, hs...)

	// 执行中间件
	c.resetHandlers()
	res := c.Next()

	// 渲染
	if !c.Writer.Written() {
		if res.Code != 0 || res.Data != nil || res.Msg != "" {
			c.Render(res)
		}
	}

}

// SetTrustedProxies 设置可信代理
func (e *Engine) SetTrustedProxies(proxies []string) error {
	e.trustedProxies = make([]*net.IPNet, 0, len(proxies))
	for _, proxy := range proxies {
		if !strings.Contains(proxy, "/") {
			if strings.Contains(proxy, ":") {
				proxy += "/128"
			} else {
				proxy += "/32"
			}
		}

		// 转换类型
		_, ipNet, err := net.ParseCIDR(proxy)
		if err != nil {
			return err
		}
		e.trustedProxies = append(e.trustedProxies, ipNet)
	}
	return nil
}

// IsTrustedProxy 判断代理是否可信
func (e *Engine) IsTrustedProxy(ip string) bool {
	if len(e.trustedProxies) == 0 {
		return false
	}
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	for _, proxy := range e.trustedProxies {
		if proxy.Contains(parsedIP) {
			return true
		}
	}
	return false
}
