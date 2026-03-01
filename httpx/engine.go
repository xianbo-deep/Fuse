package httpx

import (
	"Fuse/core"
	"Fuse/middleware"
	"net/http"
)

type Engine struct {
	router *Router
	mws    []core.HandlerFunc
}

func New() *Engine {
	return &Engine{
		router: NewRouter(),
		mws:    make([]core.HandlerFunc, 0),
	}
}

func Default() *Engine {
	e := New()
	e.Use(middleware.Defaults()...)
	return e
}

// 注入中间件
func (e *Engine) Use(middleware ...core.HandlerFunc) {
	e.mws = append(e.mws, middleware...)
}

// 常见请求方法
func (e *Engine) Get(path string, handler core.HandlerFunc) {
	e.router.Add(core.MethodGet, path, handler)
}

func (e *Engine) Post(path string, handler core.HandlerFunc) {
	e.router.Add(core.MethodPost, path, handler)
}

func (e *Engine) Put(path string, handler core.HandlerFunc) {
	e.router.Add(core.MethodPut, path, handler)
}

func (e *Engine) Delete(path string, handler core.HandlerFunc) {
	e.router.Add(core.MethodDelete, path, handler)
}

func (e *Engine) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// 创建上下文
	c := NewCtx(request.Context())

	// 设置协议
	c.Set(core.CtxKeyProtocol, core.ProtocalHTTP)

	// 换成包装后的writer
	rw := core.NewResponseWriter(writer)

	// 读取输入
	c.Writer = rw
	c.Request = request
	c.Set(core.CtxKeyMethod, request.Method)
	c.Set(core.CtxKeyPath, request.URL.Path)

	// 根据请求路径匹配业务方法
	h, params := e.router.Match(request.Method, request.URL.Path)
	if h == nil {
		c.Render(core.Fail(core.CodeNotFound, "未找到路由"))
		return
	}

	// 记录路径参数映射表
	for k, v := range params {
		c.Set("param-"+k, v)
	}
	// 组装中间件
	hs := make([]core.HandlerFunc, 0, len(e.mws)+1)
	hs = append(hs, e.mws...)
	hs = append(hs, h)

	// 执行中间件
	c.resetHandlers(hs)
	res := c.Next()

	// 渲染
	if !c.Writer.Written() {
		if res.Code != 0 || res.Data != nil || res.Msg != "" {
			c.Render(res)
		}
	}
}
