package httpx

import (
	"context"
	"net"
	"net/http"

	"github.com/xianbo-deep/Fuse/core"
	"github.com/xianbo-deep/Fuse/mux"
)

// Driver 是 http 模块的驱动，实现了 [mux.Driver] 接口。
type Driver struct {
	engine *Engine
	server *http.Server
}

// NewDriver 返回一个 *[Driver] 实例
func NewDriver(engine *Engine) *Driver {
	return &Driver{
		engine: engine,
	}
}

// Serve 初始化 http 服务
// d.server.Serve(ln) 调用了 [mux.FakeListener] 的 Accept 方法，同时底层为每个请求开启一个协程，执行了 [Engine] 的 ServeHTTP 方法。
func (d *Driver) Serve(ln net.Listener) error {
	d.server = &http.Server{
		Handler: d.engine,
	}
	return d.server.Serve(ln)
}

// Stop 执行优雅停机。
func (d *Driver) Stop(ctx context.Context) error {
	if d.server != nil {
		return d.server.Shutdown(ctx)
	}
	return nil
}

// Match 返回协议匹配器。
func (d *Driver) Match() mux.Matcher {
	return mux.IsHTTP1
}

// Engine 暴露引擎，用于挂载路由。
func (d *Driver) Engine() *Engine { return d.engine }

// ApplyMiddlewares 在引擎上挂载中间件。
func (d *Driver) ApplyMiddlewares(mws ...core.HandlerFunc) {
	d.engine.Use(mws...)
}
