package grpcx

import (
	"Fuse/core"
	"context"
	"errors"
	"sync"

	"google.golang.org/grpc/codes"
)

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

func NewCtx(ctx context.Context) *Ctx {
	return &Ctx{
		ctx:    ctx,
		values: make(map[string]any),
	}
}

// 实现core.Ctx接口
func (c *Ctx) Context() context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ctx
}

func (c *Ctx) WithContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	c.mu.Lock()
	c.ctx = ctx
	c.mu.Unlock()
}

func (c *Ctx) Set(key string, value any) {
	c.mu.Lock()
	c.values[key] = value
	c.mu.Unlock()
}

func (c *Ctx) Get(key string) (any, bool) {
	c.mu.RLock()
	v, ok := c.values[key]
	c.mu.RUnlock()
	return v, ok
}

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

func (c *Ctx) Abort() {
	c.aborted = true
	c.index = len(c.handlers)
}

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

func (c *Ctx) Aborted() bool {
	return c.aborted
}

func (c *Ctx) Err(err error) {
	if err == nil {
		return
	}
	c.errs = append(c.errs, err)
}

func (c *Ctx) Error() error {
	if len(c.errs) == 0 {
		return nil
	}
	return c.errs[len(c.errs)-1]
}

func (c *Ctx) Errors() []error {
	out := make([]error, len(c.errs))
	copy(out, c.errs)
	return out
}

func (c *Ctx) Param(key string) string {
	return ""
}

func (c *Ctx) Query(key string) string {
	return ""
}

func (c *Ctx) Bind(data any) error {
	return errors.New("GRPC does not support Bind")
}

func (c *Ctx) Success(data any) core.Result {
	return core.Result{Code: core.CodeSuccess, Data: data}
}

func (c *Ctx) Fail(code int, msg string) core.Result {
	return core.Result{Code: code, Msg: msg}
}

func (c *Ctx) Render(result core.Result) {
	return
}

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

// 重置上下文状态 清空遗留信息
func (c *Ctx) reset() {
	c.ctx = nil
	c.aborted = false
	c.index = -1
	c.handlers = nil
	c.request = nil

	clear(c.values)
	clear(c.errs)

	c.errs = c.errs[:0]
}

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
