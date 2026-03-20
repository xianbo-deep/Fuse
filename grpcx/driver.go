package grpcx

import (
	"context"
	"net"
	"time"

	"github.com/xianbo-deep/Fuse/core"
	"github.com/xianbo-deep/Fuse/mux"
	"google.golang.org/grpc/keepalive"

	"google.golang.org/grpc"
)

type Config struct {
	MaxConnectionIdle time.Duration // 客户端空闲超时时间
	MaxConnectionAge  time.Duration // 客户端连接最长绝对存活时间
	Time              time.Duration // 连接无活动 服务端发送 Ping 保活的间隔
	Timeout           time.Duration // 客户端响应 Ping 的超时时间
}

func DefaultConfig() Config {
	return Config{
		MaxConnectionIdle: 15 * time.Minute,
		MaxConnectionAge:  2 * time.Hour,
		Time:              2 * time.Hour,
		Timeout:           20 * time.Second,
	}
}

// Driver 是GRPC模块的驱动，封装了 [Engine] 与 [grpc.Server] 的集成逻辑。
//
// 它实现了 [mux.Driver] 接口，使得 gRPC 服务能够与 Fuse 框架的多路复用器（mux）协同工作。
//
// 它负责服务的配置，启动底层Server。
type Driver struct {
	// engine 是 Fuse 框架的 gRPC 引擎，负责路由注册、中间件处理和服务发现。
	engine *Engine
	// server 是底层的 gRPC 服务器实例，实际处理 gRPC 请求和响应。
	server *grpc.Server
	// cfg gRPC服务的配置
	cfg Config
}

// NewDriver 根据传入的引擎创建一个新的 [Driver] 实例。。
func NewDriver(engine *Engine, config Config, opts ...grpc.ServerOption) *Driver {
	// 传入用户自定义拦截器
	finalOpts := append([]grpc.ServerOption{}, opts...)

	// 传入配置
	finalOpts = append(finalOpts, grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle: config.MaxConnectionIdle,
		MaxConnectionAge:  config.MaxConnectionAge,
		Time:              config.Time,
		Timeout:           config.Timeout,
	}))

	// 传入框架拦截器
	finalOpts = append(finalOpts, grpc.UnaryInterceptor(engine.unaryInterceptor()), grpc.StreamInterceptor(engine.streamInterceptor()))

	// 创建底层 Server
	server := grpc.NewServer(finalOpts...)

	// 将 Server 传入引擎
	engine.SetServer(server)
	return &Driver{
		engine: engine,
		cfg:    config,
		server: server,
	}
}

// Serve 初始化 GRPC server，并根据传入的 [net.Listener] 启动 GRPC 服务。
func (d *Driver) Serve(ln net.Listener) error {
	d.server = d.engine.Server()
	return d.server.Serve(ln)
}

// Match 返回协议适配器。
func (d *Driver) Match() mux.Matcher {
	return mux.IsHTTP2
}

// Stop 实现 GRPC 服务的优雅停机，
func (d *Driver) Stop(ctx context.Context) error {
	ch := make(chan struct{})
	go func() {
		d.server.GracefulStop()
		close(ch)
	}()

	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		d.server.Stop()
		return ctx.Err()
	}
}

// Engine 暴露引擎，用于用户挂载路由。
func (d *Driver) Engine() *Engine { return d.engine }

// ApplyMiddlewares 在引擎上挂载中间件。
func (d *Driver) ApplyMiddlewares(mws ...core.HandlerFunc) {
	d.engine.Use(mws...)
}
