package cronx

import (
	"Fuse/core"
	"context"
	"errors"
	"sync"
)

type Ctx struct {
	ctx context.Context

	values  map[string]any
	aborted bool

	index    int
	handlers []core.HandlerFunc

	errs []error
	mu   sync.RWMutex
}

func NewCtx(ctx context.Context) *Ctx {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Ctx{
		ctx:    ctx,
		values: make(map[string]any),
	}
}

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

func (c *Ctx) Aborted() bool {
	return c.aborted
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

func (c *Ctx) Set(key string, val any) {
	c.mu.Lock()
	c.values[key] = val
	c.mu.Unlock()
}

func (c *Ctx) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.values[key]
	return val, ok
}

func (c *Ctx) Render(result core.Result) {

}

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

func (c *Ctx) Success(data any) core.Result {
	return core.Result{Code: core.CodeSuccess, Data: data}
}

func (c *Ctx) Fail(code int, msg string) core.Result {
	return core.Result{Code: code, Msg: msg}
}

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

func (c *Ctx) Param(key string) string {
	return ""
}

func (c *Ctx) Query(key string) string {
	return ""
}

func (c *Ctx) Bind(v any) error {
	return errors.New("cron engine does not support request binding")
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
