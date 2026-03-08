package mux

import (
	"Fuse/core"
	"context"
	"net"
)

type Driver interface {
	// 识别流量
	Match() Matcher
	// 监听
	Serve(ln net.Listener) error
	// 优雅停机
	Stop(ctx context.Context) error
	// 使用中间件
	ApplyMiddlewares(mws ...core.HandlerFunc)
}
