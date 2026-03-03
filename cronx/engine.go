package cronx

import (
	"Fuse/core"
	"Fuse/middleware"
	"context"

	"github.com/robfig/cron/v3"
)

type Engine struct {
	cron *cron.Cron
	mws  []core.HandlerFunc
}

func New() *Engine {
	return &Engine{
		cron: cron.New(cron.WithSeconds()),
		mws:  make([]core.HandlerFunc, 0),
	}
}

func Default() *Engine {
	e := New()
	e.Use(middleware.Defaults()...)
	return e
}

func (e *Engine) Use(mws ...core.HandlerFunc) {
	e.mws = append(e.mws, mws...)
}

func (e *Engine) AddFunc(spec string, handler core.HandlerFunc) (cron.EntryID, error) {
	// 设置中间件
	hs := make([]core.HandlerFunc, 0)
	hs = append(hs, e.mws...)
	hs = append(hs, handler)

	return e.cron.AddFunc(spec, func() {
		c := NewCtx(context.Background())
		c.Set(core.CtxKeyProtocol, core.ProtocolCRON)
		c.Set(core.CtxKeyPath, spec) // cron表达式当作path记录

		c.handlers = hs
		c.index = -1
		// 执行
		c.Next()
	})
}

// 启动定时任务
func (e *Engine) Start() {
	e.cron.Start()
}

// 关闭服务
func (e *Engine) Stop() context.Context {
	return e.cron.Stop()
}
