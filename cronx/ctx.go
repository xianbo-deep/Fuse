package cronx

import (
	"context"
	"errors"
	"sync"

	"github.com/xianbo-deep/Fuse/core"
)

// Ctx 是 cronx 模块中的标准上下文。
//
// 它为定时任务提供了与 HTTP/gRPC 请求类似的上下文管理能力，包括中间件支持、
// 键值存储、错误收集和取消控制，但针对定时任务的特殊需求进行了优化。
//
// 与 HTTP/gRPC 上下文不同，CronCtx 不支持请求参数绑定（Bind）、路径参数（Param）
// 和查询参数（Query），因为这些概念不适用于定时任务执行。
type Ctx struct {
	// ctx 底层上下文
	ctx context.Context

	// values 存储键值对
	values map[string]any
	// aborted 标记是否已终止当前中间件链的执行。
	aborted bool

	// index 是当前执行的中间件在处理链中的索引位置。
	//
	// 每次调用 Next() 方法时递增，用于控制中间件的执行流程。
	index int
	// handlers 是注册的中间件处理函数链。
	handlers []core.HandlerFunc

	// errs 记录任务执行过程中收集的所有错误。
	errs []error
	// mu 是读写锁，保护对共享字段的并发访问。
	mu sync.RWMutex
}

// NewCtx 创建一个新的定时任务上下文实例。
//
// ctx: 可选的底层 context.Context，如果为 nil 将使用 context.Background()。
//
// 返回值: 初始化后的 [Ctx] 指针，values 字段已分配空映射。
func NewCtx(ctx context.Context) *Ctx {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Ctx{
		ctx:    ctx,
		values: make(map[string]any),
	}
}

// Context 返回底层的上下文。
//
// 使用读锁保证线程安全。
func (c *Ctx) Context() context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ctx
}

// WithContext 替换底层的上下文。
//
// 若传入上下文为空，使用默认的空上下文。
//
// 使用写锁保证线程安全。
func (c *Ctx) WithContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	c.mu.Lock()
	c.ctx = ctx
	c.mu.Unlock()
}

// Aborted 返回当前中间件链是否已被终止。
func (c *Ctx) Aborted() bool {
	return c.aborted
}

// Next 执行中间件链中的下一个处理函数。
//
// 返回值：下一个中间件或任务处理函数返回的 [core.Result]。
func (c *Ctx) Next() core.Result {
	c.index++
	if c.index < len(c.handlers) {
		if c.aborted {
			return core.Result{}
		}
		return c.handlers[c.index](c)
	}
	return core.Result{}
}

// Abort 终止当前中间件链的执行。
func (c *Ctx) Abort() {
	c.aborted = true
	c.index = len(c.handlers)
}

// Set 在当前上下文中存储一个键值对。
//
// key: 存储的键名。
//
// val: 存储的值，可以是任意类型。
func (c *Ctx) Set(key string, val any) {
	c.mu.Lock()
	c.values[key] = val
	c.mu.Unlock()
}

// Get 从当前上下文中获取已存储的值。
//
// key: 要获取的键名。
//
// 返回值: 存储的值和是否存在标志。如果键不存在，第二个返回值为 false。
func (c *Ctx) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.values[key]
	return val, ok
}

// Render 渲染并返回输出结果。
//
// 在 cronx 模块中这里采用空实现，因为是定时任务，无需对结果进行渲染。
func (c *Ctx) Render(result core.Result) {

}

// Copy 创建当前上下文的一个深拷贝副本。
//
// 在执行异步任务需要传递上下文时，需要调用此方法生成一个上下文副本，保证线程安全。
func (c *Ctx) Copy() core.Ctx {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cp := &Ctx{
		ctx:      c.ctx,
		values:   make(map[string]any),
		aborted:  c.aborted,
		handlers: c.handlers,
		index:    c.index,
	}
	for k, v := range c.values {
		cp.values[k] = v
	}
	if c.errs != nil {
		cp.errs = make([]error, len(c.errs))
		copy(cp.errs, c.errs)
	}
	return cp
}

// Success 构造一个成功的响应结果
//
// data: 成功响应中携带的业务数据，可以是任意类型。
func (c *Ctx) Success(data any) core.Result {
	return core.Result{Code: core.CodeSuccess, Data: data}
}

// Fail 构造一个失败的响应结果，包含错误码和错误信息。
func (c *Ctx) Fail(code int, msg string) core.Result {
	return core.Result{Code: code, Msg: msg}
}

// FailWithError 使用 error 对象构造一个失败的响应结果。
//
// 内部对 [core.BizError] 进行类型断言，检查传入的 err 对象是否是自定义的 [core.BizError]，如果是则直接复用 err 对象的错误状态码和信息，
// 否则返回默认的错误状态码和信息。
func (c *Ctx) FailWithError(err error) core.Result {
	if err == nil {
		return c.Success(nil)
	}
	// 类型断言
	if bizErr, ok := err.(*core.BizError); ok {
		return c.Fail(bizErr.Code, bizErr.Msg)
	}
	return c.Fail(core.CodeInternal, err.Error())
}

// Param 获取路径参数。
//
// 在当前模块进行空实现。
func (c *Ctx) Param(key string) string {
	return ""
}

// Query 获取查询参数。
//
// 在当前模块进行空实现。
func (c *Ctx) Query(key string) string {
	return ""
}

// Bind 返回错误，在 cronx 模块不需要进行结构体绑定。
func (c *Ctx) Bind(v any) error {
	return errors.New("cron engine does not support request binding")
}

// Err 记录一个错误到上下文错误列表中。
func (c *Ctx) Err(err error) {
	if err == nil {
		return
	}
	c.errs = append(c.errs, err)
}

// Error 返回最后一个被记录的错误，如果没有错误则返回 nil。
func (c *Ctx) Error() error {
	if len(c.errs) == 0 {
		return nil
	}
	return c.errs[len(c.errs)-1]
}

// Errors 返回当前上下文中记录的所有错误列表。
func (c *Ctx) Errors() []error {
	out := make([]error, len(c.errs))
	copy(out, c.errs)
	return out
}
