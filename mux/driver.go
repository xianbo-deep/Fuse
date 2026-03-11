package mux

import (
	"context"
	"net"

	"github.com/xianbo-deep/Fuse/core"
)

// Driver 是多路复用器 [Multiplexer] 的驱动接口，定义了协议处理驱动的通用行为。
//
// 这个接口允许不同的协议驱动以统一的方式注册到多路复用器，实现在单个端口上同时处理多种协议的能力。
//
// 你可以通过实现 [Driver] 接口向框架注册你自己的服务，同样可以实现单端口协议分发，从而将你的服务无缝集成到 Fuse 框架中。
//
// 通过多路复用技术，Fuse 框架可以根据传入连接的初始数据动态识别并路由到相应的协议处理器。
//
// 你只需要实现这下面四个方法即可向 Fuse 注册你自己的服务。
type Driver interface {
	// Match 返回一个协议匹配器，用于识别流量所属的协议类型。
	Match() Matcher
	// Serve 启动驱动的服务，开始处理来自指定监听器的连接。
	Serve(ln net.Listener) error
	// Stop 优雅停机。
	Stop(ctx context.Context) error
	// ApplyMiddlewares 向驱动注册一个或多个中间件。
	ApplyMiddlewares(mws ...core.HandlerFunc)
}
