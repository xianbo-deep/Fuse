package grpcx

import (
	"context"
	"net"

	"github.com/xianbo-deep/Fuse/core"
	"github.com/xianbo-deep/Fuse/mux"

	"google.golang.org/grpc"
)

// Driver 是GRPC模块的驱动，封装了 [Engine] 与 [grpc.Server] 的集成逻辑。
//
// 它实现了 [mux.Driver] 接口，使得 gRPC 服务能够与 Fuse 框架的多路复用器（mux）协同工作。
type Driver struct {
	// engine 是 Fuse 框架的 gRPC 引擎，负责路由注册、中间件处理和服务发现。
	engine *Engine
	// server 是底层的 gRPC 服务器实例，实际处理 gRPC 请求和响应。
	server *grpc.Server
}

// NewDriver 根据传入的引擎创建一个新的 [Driver] 实例。。
func NewDriver(engine *Engine) *Driver {
	return &Driver{
		engine: engine,
	}
}

// Serve 初始化 GRPC server，并根据传入的 [net.Listener] 启动 GRPC 服务。
func (d *Driver) Serve(ln net.Listener) error {
	d.server = grpc.NewServer()
	return d.server.Serve(ln)
}

// Match 返回协议适配器。
func (d *Driver) Match() mux.Matcher {
	return mux.IsHTTP2
}

// Stop 实现 GRPC 服务的优雅停机，
func (d *Driver) Stop(ctx context.Context) error {
	d.server.GracefulStop()
	return nil
}

// Engine 暴露引擎，用于用户挂载路由。
func (d *Driver) Engine() *Engine { return d.engine }

// ApplyMiddlewares 在引擎上挂载中间件。
func (d *Driver) ApplyMiddlewares(mws ...core.HandlerFunc) {
	d.engine.Use(mws...)
}
