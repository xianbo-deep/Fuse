package fuse

import (
	"Fuse/core"
	"Fuse/cronx"
	"Fuse/grpcx"
	"Fuse/httpx"
	"Fuse/middleware"
	"Fuse/mux"
	"Fuse/wsx"
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"time"
)

const (
	CodeSuccess      = 0
	CodeBadRequest   = 1001
	CodeUnauthorized = 2001
	CodeForbidden    = 3001
	CodeNotFound     = 4004
	CodeInternal     = 9001
)

type Context = core.Ctx
type HandlerFunc = core.HandlerFunc
type Result = core.Result
type H = core.H
type BizError = core.BizError
type WsContext = wsx.WsContext

var NewError = core.NewError

type Fuse struct {
	// 引擎
	drivers map[string]mux.Driver
	// 定时任务
	cronEngine *cronx.Engine
	// 全局中间件
	mws []core.HandlerFunc
}

func New() *Fuse {
	mws := make([]core.HandlerFunc, 0)
	mws = append(mws, middleware.Defaults()...)
	return &Fuse{
		cronEngine: cronx.New(),
		drivers:    make(map[string]mux.Driver, 0),
		mws:        mws,
	}
}

func Default() *Fuse {
	f := New()
	f.Register("http", httpx.NewDriver(httpx.New()), false)
	f.Register("grpc", grpcx.NewDriver(grpcx.New()), false)
	return f
}

func RunWithMws() *Fuse {
	f := Default()
	for _, d := range f.drivers {
		d.ApplyMiddlewares(f.mws...)
	}
	return f
}

// 挂载中间件
func (fs *Fuse) Use(mws ...core.HandlerFunc) {
	fs.mws = append(fs.mws, mws...)

	// 下发给底层驱动
	for _, d := range fs.drivers {
		d.ApplyMiddlewares(mws...)
	}
}

// 返回引擎
func (fs *Fuse) HTTP() *httpx.Engine {
	if d, ok := fs.drivers["http"].(*httpx.Driver); ok {
		return d.Engine()
	}
	return nil
}

func (fs *Fuse) GRPC() *grpcx.Engine {
	if d, ok := fs.drivers["grpc"].(*grpcx.Driver); ok {
		return d.Engine()
	}
	return nil
}
func (fs *Fuse) CRON() *cronx.Engine {
	return fs.cronEngine
}

// 返回Driver
func (fs *Fuse) Driver(name string) mux.Driver {
	return fs.drivers[name]
}

// 启动服务
func (fs *Fuse) Run(addr string) error {
	if addr == "" {
		addr = ":8080"
	}
	// 监听端口
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	// 获取分发器
	muxer := mux.NewMultiplexer(ln.Addr())

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

func (fs *Fuse) Register(name string, driver mux.Driver, applyGlobalMws bool) {
	// 将已有中间件给到新驱动
	if applyGlobalMws && len(fs.mws) > 0 {
		driver.ApplyMiddlewares(fs.mws...)
	}
	fs.drivers[name] = driver
}

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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
