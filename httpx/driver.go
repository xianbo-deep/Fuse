package httpx

import (
	"Fuse/core"
	"Fuse/mux"
	"context"
	"net"
	"net/http"
)

type Driver struct {
	engine *Engine
	server *http.Server
}

func NewDriver(engine *Engine) *Driver {
	return &Driver{
		engine: engine,
	}
}

func (d *Driver) Serve(ln net.Listener) error {
	d.server = &http.Server{
		Handler: d.engine,
	}
	return d.server.Serve(ln)
}

func (d *Driver) Stop(ctx context.Context) error {
	if d.server != nil {
		return d.server.Shutdown(ctx)
	}
	return nil
}

func (d *Driver) Match() mux.Matcher {
	return mux.IsHTTP1
}

// 暴露引擎 用于挂载路由
func (d *Driver) Engine() *Engine { return d.engine }

func (d *Driver) ApplyMiddlewares(mws ...core.HandlerFunc) {
	d.engine.Use(mws...)
}
