package grpcx

import (
	"context"
	"errors"
	"sync"

	"github.com/xianbo-deep/Fuse/core"

	"google.golang.org/grpc/codes"
)

// Ctx 是 grpcx 模块的默认上下文。
//
// 实现了 [core.Ctx] 接口。
type Ctx struct {
	ctx context.Context

	// 原生请求对象
	request any

	// 共享数据
	values  map[string]any
	aborted bool

	// 中间件控制
	handlers []core.HandlerFunc
	index    int

	errs []error
	mu   sync.RWMutex
}

// NewCtx 创建一个新的GRPC上下文实例。
//
// ctx: 可选的底层 context.Context，如果为 nil 将使用 context.Background()。
//
// 返回值: 初始化后的 [Ctx] 指针，values 字段已分配空映射。
func NewCtx(ctx context.Context) *Ctx {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Ctx{
		ctx:      ctx,
		values:   make(map[string]any),
		handlers: make([]core.HandlerFunc, 0, 64),
	}
}

// 实现core.Ctx接口

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

// Set 在当前上下文中存储一个键值对。
//
// key: 存储的键名。
//
// val: 存储的值，可以是任意类型。
func (c *Ctx) Set(key string, value any) {
	c.mu.Lock()
	c.values[key] = value
	c.mu.Unlock()
}

// Get 从当前上下文中获取已存储的值。
//
// key: 要获取的键名。
//
// 返回值: 存储的值和是否存在标志。如果键不存在，第二个返回值为 false。
func (c *Ctx) Get(key string) (any, bool) {
	c.mu.RLock()
	v, ok := c.values[key]
	c.mu.RUnlock()
	return v, ok
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

// Copy 创建当前上下文的一个深拷贝副本。
//
// 在执行异步任务需要传递上下文时，需要调用此方法生成一个上下文副本，保证线程安全。
func (c *Ctx) Copy() core.Ctx {
	cp := &Ctx{
		ctx:      c.ctx,
		request:  c.request,
		values:   make(map[string]any),
		aborted:  c.aborted,
		handlers: c.handlers,
		index:    c.index,
		errs:     c.errs,
	}
	for k, v := range c.values {
		cp.values[k] = v
	}
	if c.errs != nil {
		cp.errs = make([]error, 0, len(c.errs))
		copy(cp.errs, c.errs)
	}
	return cp
}

// Aborted 返回当前中间件链是否已被终止。
func (c *Ctx) Aborted() bool {
	return c.aborted
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

// Bind 返回错误，目前未实现。
func (c *Ctx) Bind(data any) error {
	return errors.New("GRPC does not support Bind")
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

// Render 进行协议响应头和响应体的渲染，目前是空实现，
func (c *Ctx) Render(result core.Result) {
	return
}

// FailWithError 使用 error 对象构造一个失败的响应结果。
//
// 内部对 [core.BizError] 进行类型断言，检查传入的 err 对象是否是自定义的 [core.BizError]，如果是则直接复用 err 对象的错误状态码和信息，
// 否则返回默认的错误状态码和信息。
func (c *Ctx) FailWithError(err error) core.Result {
	if err == nil {
		return c.Success(nil)
	}
	if bizErr, ok := err.(*core.BizError); ok {
		res := c.Fail(bizErr.Code, bizErr.Msg)
		if bizErr.GrpcStatus != 0 {
			res = res.WithGrpcStatus(bizErr.GrpcStatus)
		}
		return res
	}
	return c.Fail(core.CodeInternal, err.Error()).WithGrpcStatus(int(codes.Internal))
}

// reset 重置上下文状态，清空遗留信息，为上下文复用做准备。
func (c *Ctx) reset() {
	c.ctx = nil
	c.aborted = false
	c.index = -1
	c.request = nil

	clear(c.values)
	clear(c.handlers)
	clear(c.errs)

	c.handlers = c.handlers[:0]
	c.errs = c.errs[:0]
}

// grpcCodeFromBizCode 业务状态码与GRPC状态码的映射。
func grpcCodeFromBizCode(code int) codes.Code {
	switch code {
	case core.CodeBadRequest:
		return codes.InvalidArgument
	case core.CodeUnauthorized:
		return codes.Unauthenticated
	case core.CodeForbidden:
		return codes.PermissionDenied
	case core.CodeNotFound:
		return codes.NotFound
	case core.CodeInternal:
		return codes.Internal
	default:
		return codes.Unknown
	}
}
