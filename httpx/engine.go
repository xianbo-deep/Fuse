package httpx

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/xianbo-deep/Fuse/core"
	"github.com/xianbo-deep/Fuse/middleware"
)

type Engine struct {
	router *Router
	pool   sync.Pool
	*RouterGroup
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
		c := NewCtx(context.Background())
		return c
	}
	return e
}

func Default() *Engine {
	e := New()
	e.Use(middleware.Defaults()...)
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
