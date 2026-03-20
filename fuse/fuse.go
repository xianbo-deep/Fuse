// Package fuse 是 Fuse 框架的统一入口和核心门面。
//
// Fuse 是一个轻量级、协议无关的 Go 服务端框架骨架。该包旨在为开发者提供简洁一致的 API，
// 屏蔽底层不同协议的复杂依赖关系。
//
// 主要特性包括：
//   - 统一上下文：暴露了统一的 [Context]、[Result] 和 [HandlerFunc] 类型别名，实现跨协议范式统一。
//   - 协议多路复用：通过 [Fuse.Run] 启动服务，支持在单个端口上同时处理 HTTP/1.1 和 HTTP/2 (gRPC) 流量。
//   - 引擎管理：提供 [Fuse.HTTP]、[Fuse.GRPC] 和 [Fuse.CRON] 等方法快速获取对应协议的底层执行引擎。
//   - 全局中间件：通过 [Fuse.Use] 注册的中间件会自动下发应用到所有已注册的协议驱动中。
//   - 驱动注册与获取：支持用户通过 [Fuse.Register] 与 [Fuse.Driver] 进行自定义驱动的注册与获取。
//
// 简单示例:
//
//	func main() {
//	    app := fuse.Default()
//
//	    app.HTTP().GET("/ping", func(c fuse.Context) fuse.Result {
//	        return c.Success(fuse.H{"message": "pong"})
//	    })
//
//	    if err := app.Run(":8080"); err != nil {
//	        panic(err)
//	    }
//	}
package fuse

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/xianbo-deep/Fuse/core"
	"github.com/xianbo-deep/Fuse/cronx"
	"github.com/xianbo-deep/Fuse/grpcx"
	"github.com/xianbo-deep/Fuse/httpx"
	"github.com/xianbo-deep/Fuse/middleware"
	"github.com/xianbo-deep/Fuse/mux"
	"github.com/xianbo-deep/Fuse/ssex"
	"github.com/xianbo-deep/Fuse/wsx"
)

const (
	CodeSuccess      = 0
	CodeBadRequest   = 1001
	CodeUnauthorized = 2001
	CodeForbidden    = 3001
	CodeNotFound     = 4004
	CodeInternal     = 9001
)

// Context 请求上下文类型别名
type Context = core.Ctx

// HandlerFunc 处理器函数类型别名
type HandlerFunc = core.HandlerFunc

// Result 统一响应结果类型别名
type Result = core.Result

// H Map类型别名，用于构建响应数据
type H = core.H

// BizError 业务错误类型别名
type BizError = core.BizError

// WsContext WebSocket上下文类型别名
type WsContext = wsx.WsContext

// Stream SSE上下文类型别名
type Stream = ssex.Stream

// NewError 创建业务错误的快捷函数
var NewError = core.NewError

// Fuse 应用框架主结构体
// 支持多协议、定时任务、中间件等功能
type Fuse struct {
	// 引擎
	drivers map[string]mux.Driver
	// 定时任务
	cronEngine *cronx.Engine
	// 全局中间件
	mws []core.HandlerFunc
	// 全局配置
	config *core.Config
}

// New 返回 [Fuse] 实例。
func New(cfg ...*core.Config) *Fuse {
	var config = core.DefaultConfig()
	if len(cfg) > 0 {
		config = cfg[0]
	}
	return &Fuse{
		cronEngine: cronx.New(),
		drivers:    make(map[string]mux.Driver, 0),
		mws:        make([]core.HandlerFunc, 0),
		config:     config,
	}
}

// Default 创建默认配置的Fuse实例
//
// 包含HTTP和gRPC驱动
func Default(cfg ...*core.Config) *Fuse {
	f := New(cfg...)
	f.Register("http", httpx.NewDriver(httpx.New(), httpx.DefaultConfig()), false)
	f.Register("grpc", grpcx.NewDriver(grpcx.New(), grpcx.DefaultConfig()), false)
	f.Use(middleware.Defaults()...)
	return f
}

// Use 添加全局中间件
//
// 中间件会按添加顺序执行，并自动应用到所有已注册驱动
func (fs *Fuse) Use(mws ...core.HandlerFunc) {
	fs.mws = append(fs.mws, mws...)

	// 下发给底层驱动
	for _, d := range fs.drivers {
		d.ApplyMiddlewares(mws...)
	}
}

// HTTP 获取HTTP引擎实例
func (fs *Fuse) HTTP() *httpx.Engine {
	if d, ok := fs.drivers["http"].(*httpx.Driver); ok {
		return d.Engine()
	}
	return nil
}

// GRPC 获取gRPC引擎实例
func (fs *Fuse) GRPC() *grpcx.Engine {
	if d, ok := fs.drivers["grpc"].(*grpcx.Driver); ok {
		return d.Engine()
	}
	return nil
}

// CRON 获取定时任务引擎实例
func (fs *Fuse) CRON() *cronx.Engine {
	return fs.cronEngine
}

// Driver 根据名称获取驱动实例
func (fs *Fuse) Driver(name string) mux.Driver {
	return fs.drivers[name]
}

// Run 启动Fuse框架服务
//
// 功能: 1.监听端口 2.启动多路复用器 3.启动所有驱动 4.启动定时任务 5.优雅停机
func (fs *Fuse) Run(addr string) error {
	printBanner()
	if addr == "" {
		addr = fs.config.Port
	}
	// 监听端口
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	// 获取分发器
	muxer := mux.NewMultiplexer(ln.Addr(), fs.config.MuxHandShakeTimeout)

	// 启动服务
	for name, driver := range fs.drivers {
		listener := muxer.Match(driver.Match())
		if listener != nil {
			go func(name string, driver mux.Driver, listener net.Listener) {
				log.Printf("[FUSE] Driver [%s] is starting...", name)
				if err := driver.Serve(listener); err != nil {
					log.Printf("[FUSE] Driver [%s] error: %v", name, err)
				}
			}(name, driver, listener)
		}
	}
	// 启动分发器进行协议分发
	go muxer.ServeLoop(ln)

	// 启动定时任务
	fs.cronEngine.Start()

	return fs.gracefulStop(ln)
}

func printBanner() {
	const colorCyan = "\033[1;36m"
	const colorReset = "\033[0m"

	banner := `
    F U S E  |  Multi Protocol Framework
    ----------------------------------------------
     ███████╗██╗   ██╗███████╗███████╗
     ██╔════╝██║   ██║██╔════╝██╔════╝
     █████╗  ██║   ██║███████╗█████╗  
     ██╔══╝  ██║   ██║╚════██║██╔══╝  
     ██║     ╚██████╔╝███████║███████╗
     ╚═╝      ╚═════╝ ╚══════╝╚══════╝
    `
	fmt.Println(colorCyan + banner + colorReset)
	fmt.Printf("Fuse Framework starting\n")
}

// Register 注册新的协议驱动
//
// 注册时可选是否自动应用已配置的全局中间件
func (fs *Fuse) Register(name string, driver mux.Driver, applyGlobalMws bool) {
	// 将已有中间件给到新驱动
	if applyGlobalMws && len(fs.mws) > 0 {
		driver.ApplyMiddlewares(fs.mws...)
	}
	fs.drivers[name] = driver
}

// gracefulStop 优雅停机
func (fs *Fuse) gracefulStop(ln net.Listener) error {
	// 优雅停机
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	// 阻塞等待
	<-quit

	// 关闭监听服务
	if ln != nil {
		ln.Close()
	}
	// 关闭服务
	ctx, cancel := context.WithTimeout(context.Background(), fs.config.StopTimeout)
	defer cancel()

	for name, driver := range fs.drivers {
		log.Printf("[FUSE] Driver [%s] is stopping...", name)
		if err := driver.Stop(ctx); err != nil {
			log.Printf("[FUSE] Driver [%s] error: %v", name, err)
		}
	}

	if fs.cronEngine != nil {
		cronCtx := fs.cronEngine.Stop()
		select {
		case <-cronCtx.Done():
			log.Printf("[FUSE] CRON engine stopped")
		case <-ctx.Done():
			log.Printf("[FUSE] CRON engine stop timeout")
		}
	}

	return nil
}
