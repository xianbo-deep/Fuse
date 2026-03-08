package grpcx

import (
	"Fuse/core"
	"Fuse/mux"
	"context"
	"net"

	"google.golang.org/grpc"
)

type Driver struct {
	engine *Engine
	server *grpc.Server
}

func NewDriver(engine *Engine) *Driver {
	return &Driver{
		engine: engine,
	}
}
func (d *Driver) Serve(ln net.Listener) error {
	d.server = grpc.NewServer()
	return d.server.Serve(ln)
}

func (d *Driver) Match() mux.Matcher {
	return mux.IsHTTP2
}

func (d *Driver) Stop(ctx context.Context) error {
	d.server.GracefulStop()
	return nil
}

// 暴露引擎 用于用户挂载路由
func (d *Driver) Engine() *Engine { return d.engine }

func (d *Driver) ApplyMiddlewares(mws ...core.HandlerFunc) {
	d.engine.Use(mws...)
}
