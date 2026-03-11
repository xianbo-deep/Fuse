package cronx

import (
	"context"

	"github.com/xianbo-deep/Fuse/core"
	"github.com/xianbo-deep/Fuse/middleware"

	"github.com/robfig/cron/v3"
)

// Engine 是 cronx 模块的定时任务引擎，基于 robfig/cron/v3 封装。
//
// 引擎支持标准的 cron 表达式格式（秒 分 时 日 月 周）和链式中间件执行模型。
type Engine struct {
	// cron 是底层的 robfig/cron 实例，负责实际的定时调度和执行。
	cron *cron.Cron

	// mws 是注册到引擎的中间件链，会应用于所有通过此引擎添加的定时任务。
	mws []core.HandlerFunc
}

// New 初始化一个无中间件的引擎。
//
// 使用秒级精度的 cron 。
func New() *Engine {
	return &Engine{
		cron: cron.New(cron.WithSeconds()),
		mws:  make([]core.HandlerFunc, 0),
	}
}

// Default 使用默认的中间件初始化引擎。
func Default() *Engine {
	e := New()
	e.Use(middleware.Defaults()...)
	return e
}

// Use 给当前引擎挂载中间件。
//
// 可传递一个或多个中间件。
func (e *Engine) Use(mws ...core.HandlerFunc) {
	e.mws = append(e.mws, mws...)
}

// AddFunc 挂载定时任务到当前的引擎。
//
// spec: cron表达式
//
// handler: 用户定义的定时任务，具体类型前往 [core.HandlerFunc] 查看。
//
// 返回任务ID与错误。
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

// Start 启动定时任务。
func (e *Engine) Start() {
	e.cron.Start()
}

// Stop 关闭定时任务。
func (e *Engine) Stop() context.Context {
	return e.cron.Stop()
}
