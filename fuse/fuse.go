package fuse

import (
	"Fuse/core"
	"Fuse/cronx"
	"Fuse/grpcx"
	"Fuse/httpx"
	"net"
	"net/http"
	"sync"
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

var NewError = core.NewError

type Fuse struct {
	// 引擎
	httpEngine *httpx.Engine
	grpcEngine *grpcx.Engine
	cronEngine *cronx.Engine

	// 全局中间件
	mws []core.HandlerFunc
}

func New() *Fuse {
	return &Fuse{
		httpEngine: httpx.New(),
		grpcEngine: grpcx.New(),
		cronEngine: cronx.New(),
	}
}

func (fs *Fuse) Default() *Fuse {
	return &Fuse{
		httpEngine: httpx.Default(),
		grpcEngine: grpcx.Default(),
		cronEngine: cronx.Default(),
	}
}

// 挂载中间件
func (fs *Fuse) Use(mws ...core.HandlerFunc) {
	fs.mws = append(fs.mws, mws...)

	// 下发给底层引擎
	fs.httpEngine.Use(mws...)
	fs.grpcEngine.Use(mws...)
	fs.cronEngine.Use(mws...)
}

// 返回引擎
func (fs *Fuse) HTTP() *httpx.Engine {
	return fs.httpEngine
}

func (fs *Fuse) GRPC() *grpcx.Engine {
	return fs.grpcEngine
}
func (fs *Fuse) CRON() *cronx.Engine {
	return fs.cronEngine
}

// 启动服务
func (fs *Fuse) Run(httpAddr string, grpcAddr string) error {
	var wg sync.WaitGroup
	if httpAddr != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = http.ListenAndServe(httpAddr, fs.httpEngine)
		}()
	}

	if grpcAddr != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lis, err := net.Listen("tcp", grpcAddr)
			if err != nil {
				panic(err) // 监听端口失败直接报错
			}
			_ = fs.grpcEngine.Server().Serve(lis)
		}()
	}
	// 启动定时任务
	fs.cronEngine.Start()
	wg.Wait()

	return nil
}
